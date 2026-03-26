# zschema sub-schema for zgrab2's minecraft-bedrock module
# Registers zgrab2-minecraft-bedrock globally, and minecraft-bedrock with the main zgrab2 schema.
from zschema.leaves import *
from zschema.compounds import *
import zschema.registry

from . import zgrab2

# modules/minecraft_bedrock/scanner.go - Results
minecraft_bedrock_scan_response = SubRecord(
    {
        "result": SubRecord(
            {
                "edition": String(
                    doc="The server edition identifier.",
                    examples=["MCPE", "MCEE"],
                ),
                "motd": String(
                    doc="The server's primary message of the day (MOTD).",
                ),
                "protocol": Signed32BitInteger(
                    doc="The Bedrock protocol version number.",
                ),
                "version": String(
                    doc="The Minecraft Bedrock version string.",
                    examples=["1.20.1", "1.19.80"],
                ),
                "online_players": Signed32BitInteger(
                    doc="The number of players currently online.",
                ),
                "max_players": Signed32BitInteger(
                    doc="The maximum number of players allowed.",
                ),
                "server_id": String(
                    doc="The server's unique identifier string.",
                ),
                "motd2": String(
                    doc="The server's secondary MOTD (typically the world name).",
                ),
                "game_mode": String(
                    doc="The server's game mode.",
                    examples=["Survival", "Creative", "Adventure"],
                ),
                "game_mode_num": Signed32BitInteger(
                    doc="The numeric game mode identifier.",
                ),
                "port_ipv4": Signed32BitInteger(
                    doc="The server's IPv4 port.",
                ),
                "port_ipv6": Signed32BitInteger(
                    doc="The server's IPv6 port.",
                ),
                "server_guid": String(
                    doc="The server's RakNet GUID.",
                ),
                "raw_response": String(
                    doc="The raw semicolon-delimited status string (only in verbose mode).",
                ),
            }
        )
    },
    extends=zgrab2.base_scan_response,
)

zschema.registry.register_schema(
    "zgrab2-minecraft-bedrock", minecraft_bedrock_scan_response
)

zgrab2.register_scan_response_type(
    "minecraft-bedrock", minecraft_bedrock_scan_response
)
