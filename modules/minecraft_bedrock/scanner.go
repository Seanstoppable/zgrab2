// Package minecraft_bedrock implements the Minecraft Bedrock Edition server
// ping protocol for zgrab2. It sends a RakNet Unconnected Ping over UDP and
// parses the Unconnected Pong response to extract server status information.
package minecraft_bedrock

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/zmap/zgrab2"
)

// rakNetMagic is the 16-byte offline message identifier used in RakNet
// Unconnected Ping/Pong packets.
var rakNetMagic = []byte{
	0x00, 0xff, 0xff, 0x00, 0xfe, 0xfe, 0xfe, 0xfe,
	0xfd, 0xfd, 0xfd, 0xfd, 0x12, 0x34, 0x56, 0x78,
}

const (
	unconnectedPingID = 0x01
	unconnectedPongID = 0x1C
	// Minimum pong size: 1 (ID) + 8 (timestamp) + 8 (server GUID) + 16 (magic) + 2 (data length) = 35
	minPongSize = 35
)

// Flags holds the command-line flags for the minecraft-bedrock module.
type Flags struct {
	zgrab2.BaseFlags `group:"Basic Options"`
}

// Module implements the zgrab2.ScanModule interface.
type Module struct{}

// Scanner implements the zgrab2.Scanner interface.
type Scanner struct {
	config            *Flags
	dialerGroupConfig *zgrab2.DialerGroupConfig
}

// Results is the JSON-serializable output of a Minecraft Bedrock scan.
type Results struct {
	Edition       string `json:"edition,omitempty"`
	MOTD          string `json:"motd,omitempty"`
	Protocol      int    `json:"protocol,omitempty"`
	Version       string `json:"version,omitempty"`
	OnlinePlayers int    `json:"online_players,omitempty"`
	MaxPlayers    int    `json:"max_players,omitempty"`
	ServerID      string `json:"server_id,omitempty"`
	MOTD2         string `json:"motd2,omitempty"`
	GameMode      string `json:"game_mode,omitempty"`
	GameModeNum   int    `json:"game_mode_num,omitempty"`
	PortIPv4      int    `json:"port_ipv4,omitempty"`
	PortIPv6      int    `json:"port_ipv6,omitempty"`
	ServerGUID    uint64 `json:"server_guid,omitempty"`
	RawResponse   string `json:"raw_response,omitempty"`

	rawStatusString string // internal; not serialized
}

// RegisterModule registers the minecraft-bedrock module with the zgrab2 framework.
func RegisterModule() {
	var m Module
	_, err := zgrab2.AddCommand("minecraft-bedrock", "Minecraft Bedrock Edition Ping", m.Description(), 19132, &m)
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
	return "Probe for Minecraft Bedrock Edition servers using RakNet Unconnected Ping to retrieve version, player count, and server description"
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
		TransportAgnosticDialerProtocol: zgrab2.TransportUDP,
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
	return "minecraft-bedrock"
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

	// Send Unconnected Ping
	ping := buildUnconnectedPing()
	if _, err := conn.Write(ping); err != nil {
		return zgrab2.TryGetScanStatus(err), nil, fmt.Errorf("failed to send ping: %w", err)
	}

	// Read Unconnected Pong
	buf := make([]byte, 1500) // MTU-sized buffer for single UDP datagram
	n, err := conn.Read(buf)
	if err != nil {
		return zgrab2.TryGetScanStatus(err), nil, fmt.Errorf("failed to read pong: %w", err)
	}

	results, err := parseUnconnectedPong(buf[:n])
	if err != nil {
		return zgrab2.SCAN_PROTOCOL_ERROR, nil, fmt.Errorf("failed to parse pong: %w", err)
	}

	if scanner.config.Verbose {
		results.RawResponse = results.rawStatusString
	}
	results.rawStatusString = "" // Don't leak internal field

	return zgrab2.SCAN_SUCCESS, results, nil
}

// buildUnconnectedPing constructs a 33-byte RakNet Unconnected Ping packet.
func buildUnconnectedPing() []byte {
	packet := make([]byte, 33)
	packet[0] = unconnectedPingID
	// Ping timestamp (8 bytes, big-endian)
	binary.BigEndian.PutUint64(packet[1:9], uint64(time.Now().UnixMilli()))
	// RakNet offline magic (16 bytes)
	copy(packet[9:25], rakNetMagic)
	// Client GUID (8 bytes, random)
	binary.BigEndian.PutUint64(packet[25:33], rand.Uint64())
	return packet
}

// parseUnconnectedPong parses a RakNet Unconnected Pong packet into Results.
func parseUnconnectedPong(data []byte) (*Results, error) {
	if len(data) < minPongSize {
		return nil, fmt.Errorf("pong too short: %d bytes (minimum %d)", len(data), minPongSize)
	}

	if data[0] != unconnectedPongID {
		return nil, fmt.Errorf("unexpected packet ID: 0x%02X (expected 0x%02X)", data[0], unconnectedPongID)
	}

	// Parse server GUID at offset 9
	serverGUID := binary.BigEndian.Uint64(data[9:17])

	// Data length at offset 33
	dataLen := int(binary.BigEndian.Uint16(data[33:35]))
	if len(data) < 35+dataLen {
		return nil, fmt.Errorf("pong data truncated: expected %d bytes, have %d", dataLen, len(data)-35)
	}

	statusString := string(data[35 : 35+dataLen])

	results := &Results{
		ServerGUID:      serverGUID,
		rawStatusString: statusString,
	}

	// Parse semicolon-delimited fields:
	// MCPE;<MOTD>;<protocol>;<version>;<online>;<max>;<serverID>;<MOTD2>;<gamemode>;<gamemodeNum>;<port4>;<port6>
	fields := strings.Split(statusString, ";")

	if len(fields) >= 1 {
		results.Edition = fields[0]
	}
	if len(fields) >= 2 {
		results.MOTD = fields[1]
	}
	if len(fields) >= 3 {
		results.Protocol, _ = strconv.Atoi(fields[2])
	}
	if len(fields) >= 4 {
		results.Version = fields[3]
	}
	if len(fields) >= 5 {
		results.OnlinePlayers, _ = strconv.Atoi(fields[4])
	}
	if len(fields) >= 6 {
		results.MaxPlayers, _ = strconv.Atoi(fields[5])
	}
	if len(fields) >= 7 {
		results.ServerID = fields[6]
	}
	if len(fields) >= 8 {
		results.MOTD2 = fields[7]
	}
	if len(fields) >= 9 {
		results.GameMode = fields[8]
	}
	if len(fields) >= 10 {
		results.GameModeNum, _ = strconv.Atoi(fields[9])
	}
	if len(fields) >= 11 {
		results.PortIPv4, _ = strconv.Atoi(fields[10])
	}
	if len(fields) >= 12 {
		results.PortIPv6, _ = strconv.Atoi(fields[11])
	}

	return results, nil
}
