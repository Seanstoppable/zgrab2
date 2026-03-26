// Package minecraft implements the Minecraft Server List Ping (SLP) protocol
// for zgrab2. It connects to Minecraft Java Edition servers and retrieves
// server status information including version, player count, and description.
package minecraft

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"

	log "github.com/sirupsen/logrus"

	"github.com/zmap/zgrab2"
)

// Flags holds the command-line flags for the minecraft module.
type Flags struct {
	zgrab2.BaseFlags `group:"Basic Options"`

	ProtocolVersion int  `long:"protocol-version" default:"763" description:"Minecraft protocol version to advertise in handshake (763 = 1.20.1)"`
	IncludeFavicon  bool `long:"include-favicon" description:"Include the server favicon (base64 PNG) in the output"`
}

// Module implements the zgrab2.ScanModule interface.
type Module struct{}

// Scanner implements the zgrab2.Scanner interface.
type Scanner struct {
	config            *Flags
	dialerGroupConfig *zgrab2.DialerGroupConfig
}

// Results is the JSON-serializable output of a Minecraft SLP scan.
type Results struct {
	Version     *ServerVersion `json:"version,omitempty"`
	Players     *ServerPlayers `json:"players,omitempty"`
	Description string         `json:"description,omitempty"`
	Favicon     string         `json:"favicon,omitempty"`
	RawResponse string         `json:"raw_response,omitempty"`
}

// ServerVersion contains the Minecraft version info returned by the server.
type ServerVersion struct {
	Name     string `json:"name"`
	Protocol int    `json:"protocol"`
}

// ServerPlayers contains the player count info returned by the server.
type ServerPlayers struct {
	Max    int `json:"max"`
	Online int `json:"online"`
}

// serverStatusResponse is used internally for unmarshalling the server's JSON response.
// The Description field uses json.RawMessage because servers may return either
// a plain string or a chat component object {"text": "..."}.
type serverStatusResponse struct {
	Version     *ServerVersion  `json:"version"`
	Players     *ServerPlayers  `json:"players"`
	Description json.RawMessage `json:"description"`
	Favicon     string          `json:"favicon"`
}

// RegisterModule registers the minecraft module with the zgrab2 framework.
func RegisterModule() {
	var m Module
	_, err := zgrab2.AddCommand("minecraft", "Minecraft Server List Ping", m.Description(), 25565, &m)
	if err != nil {
		log.Fatal(err)
	}
}

func (m *Module) NewFlags() any {
	return new(Flags)
}

func (m *Module) NewScanner() zgrab2.Scanner {
	return new(Scanner)
}

func (m *Module) Description() string {
	return "Probe for Minecraft servers using the Server List Ping (SLP) protocol to retrieve version, player count, and server description"
}

func (f *Flags) Validate(_ []string) error {
	return nil
}

func (f *Flags) Help() string {
	return ""
}

func (scanner *Scanner) Init(flags zgrab2.ScanFlags) error {
	f, _ := flags.(*Flags)
	scanner.config = f
	scanner.dialerGroupConfig = &zgrab2.DialerGroupConfig{
		TransportAgnosticDialerProtocol: zgrab2.TransportTCP,
		BaseFlags:                       &f.BaseFlags,
	}
	return nil
}

func (scanner *Scanner) InitPerSender(senderID int) error {
	return nil
}

func (scanner *Scanner) GetName() string {
	return scanner.config.Name
}

func (scanner *Scanner) GetTrigger() string {
	return scanner.config.Trigger
}

func (scanner *Scanner) Protocol() string {
	return "minecraft"
}

func (scanner *Scanner) GetDialerGroupConfig() *zgrab2.DialerGroupConfig {
	return scanner.dialerGroupConfig
}

func (scanner *Scanner) GetScanMetadata() any {
	return nil
}

func (scanner *Scanner) Scan(ctx context.Context, dialGroup *zgrab2.DialerGroup, target *zgrab2.ScanTarget) (zgrab2.ScanStatus, any, error) {
	conn, err := dialGroup.Dial(ctx, target)
	if err != nil {
		return zgrab2.TryGetScanStatus(err), nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer zgrab2.CloseConnAndHandleError(conn)

	host := target.Host()
	port := scanner.config.BaseFlags.Port

	// Send handshake packet
	handshake := buildHandshakePacket(scanner.config.ProtocolVersion, host, uint16(port))
	if _, err := conn.Write(handshake); err != nil {
		return zgrab2.TryGetScanStatus(err), nil, fmt.Errorf("failed to send handshake: %w", err)
	}

	// Send status request packet
	statusReq := buildStatusRequestPacket()
	if _, err := conn.Write(statusReq); err != nil {
		return zgrab2.TryGetScanStatus(err), nil, fmt.Errorf("failed to send status request: %w", err)
	}

	// Read response
	_, data, err := readPacket(conn)
	if err != nil {
		return zgrab2.TryGetScanStatus(err), nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse the JSON string from the response data
	jsonStr, err := readString(bytes.NewReader(data))
	if err != nil {
		return zgrab2.SCAN_PROTOCOL_ERROR, nil, fmt.Errorf("failed to read JSON string from response: %w", err)
	}

	// Unmarshal into our internal type that handles the description polymorphism
	var statusResp serverStatusResponse
	if err := json.Unmarshal([]byte(jsonStr), &statusResp); err != nil {
		return zgrab2.SCAN_PROTOCOL_ERROR, nil, fmt.Errorf("failed to parse server response JSON: %w", err)
	}

	results := &Results{
		Version: statusResp.Version,
		Players: statusResp.Players,
	}

	if scanner.config.IncludeFavicon {
		results.Favicon = statusResp.Favicon
	}

	// Handle description: can be a plain string or a chat component {"text": "..."}
	results.Description = parseDescription(statusResp.Description)

	if scanner.config.Verbose {
		results.RawResponse = jsonStr
	}

	return zgrab2.SCAN_SUCCESS, results, nil
}

// parseDescription extracts a human-readable string from the Minecraft description
// field, which can be either a plain JSON string or a chat component object.
func parseDescription(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try plain string first
	var plainStr string
	if err := json.Unmarshal(raw, &plainStr); err == nil {
		return plainStr
	}

	// Try chat component object {"text": "..."}
	var chatComponent struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &chatComponent); err == nil {
		return chatComponent.Text
	}

	// Fallback: return the raw JSON
	return string(raw)
}

// --- Minecraft protocol helpers ---

// writeVarInt encodes an integer as a Minecraft protocol VarInt into the buffer.
func writeVarInt(buf *bytes.Buffer, value int) {
	uval := uint32(value)
	for {
		b := byte(uval & 0x7F)
		uval >>= 7
		if uval != 0 {
			b |= 0x80
		}
		buf.WriteByte(b)
		if uval == 0 {
			break
		}
	}
}

// readVarInt decodes a Minecraft protocol VarInt from a reader.
func readVarInt(r io.Reader) (int, error) {
	var result uint32
	var numRead uint
	buf := make([]byte, 1)
	for {
		if _, err := io.ReadFull(r, buf); err != nil {
			return 0, err
		}
		b := buf[0]
		result |= uint32(b&0x7F) << (7 * numRead)
		numRead++
		if numRead > 5 {
			return 0, fmt.Errorf("VarInt too long")
		}
		if b&0x80 == 0 {
			break
		}
	}
	return int(result), nil
}

// readString reads a VarInt-length-prefixed UTF-8 string.
func readString(r io.Reader) (string, error) {
	length, err := readVarInt(r)
	if err != nil {
		return "", fmt.Errorf("failed to read string length: %w", err)
	}
	if length < 0 || length > 1<<20 {
		return "", fmt.Errorf("string length out of bounds: %d", length)
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return "", fmt.Errorf("failed to read string data: %w", err)
	}
	return string(data), nil
}

// buildHandshakePacket constructs the Minecraft handshake packet for status query.
func buildHandshakePacket(protocolVersion int, host string, port uint16) []byte {
	// Build packet data: PacketID(0x00) + ProtocolVersion + ServerAddress + ServerPort + NextState(1)
	var payload bytes.Buffer
	writeVarInt(&payload, 0x00) // Packet ID
	writeVarInt(&payload, protocolVersion)
	// Server address as VarInt-prefixed string
	writeVarInt(&payload, len(host))
	payload.WriteString(host)
	// Server port as unsigned short (big-endian)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, port)
	payload.Write(portBytes)
	// Next state: 1 = status
	writeVarInt(&payload, 1)

	// Frame the packet: Length prefix + payload
	var packet bytes.Buffer
	writeVarInt(&packet, payload.Len())
	packet.Write(payload.Bytes())
	return packet.Bytes()
}

// buildStatusRequestPacket constructs the empty status request packet.
func buildStatusRequestPacket() []byte {
	var packet bytes.Buffer
	writeVarInt(&packet, 1) // Length: 1 byte (just the packet ID)
	writeVarInt(&packet, 0) // Packet ID: 0x00
	return packet.Bytes()
}

// readPacket reads a single Minecraft protocol packet from the connection.
// Returns the packet ID and the remaining data bytes.
func readPacket(conn net.Conn) (int, []byte, error) {
	length, err := readVarInt(conn)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read packet length: %w", err)
	}
	if length < 0 || length > 1<<21 {
		return 0, nil, fmt.Errorf("packet length out of bounds: %d", length)
	}

	packetData := make([]byte, length)
	if _, err := io.ReadFull(conn, packetData); err != nil {
		return 0, nil, fmt.Errorf("failed to read packet data: %w", err)
	}

	reader := bytes.NewReader(packetData)
	packetID, err := readVarInt(reader)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read packet ID: %w", err)
	}

	remaining := make([]byte, reader.Len())
	if _, err := io.ReadFull(reader, remaining); err != nil {
		return 0, nil, fmt.Errorf("failed to read remaining packet data: %w", err)
	}

	return packetID, remaining, nil
}
