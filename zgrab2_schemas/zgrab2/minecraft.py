# zschema sub-schema for zgrab2's minecraft module
# Registers zgrab2-minecraft globally, and minecraft with the main zgrab2 schema.
from zschema.leaves import *
from zschema.compounds import *
import zschema.registry

from . import zgrab2

# modules/minecraft/scanner.go - Results
minecraft_scan_response = SubRecord(
    {
        "result": SubRecord(
            {
                "version": SubRecord(
                    {
                        "name": String(
                            doc="The Minecraft version name reported by the server.",
                            examples=["1.20.1", "1.19.4"],
                        ),
                        "protocol": Signed32BitInteger(
                            doc="The Minecraft protocol version number.",
                            examples=[763, 765],
                        ),
                    },
                    doc="The server's version information.",
                ),
                "players": SubRecord(
                    {
                        "max": Signed32BitInteger(
                            doc="The maximum number of players allowed on the server.",
                        ),
                        "online": Signed32BitInteger(
                            doc="The number of players currently online.",
                        ),
                    },
                    doc="The server's player count information.",
                ),
                "description": String(
                    doc="The server's description (MOTD).",
                    examples=["A Minecraft Server", "vanillaera"],
                ),
                "favicon": String(
                    doc="The server's favicon as a base64-encoded PNG data URI.",
                ),
                "raw_response": String(
                    doc="The raw JSON response from the server (only included in verbose mode).",
                ),
            }
        )
    },
    extends=zgrab2.base_scan_response,
)

zschema.registry.register_schema("zgrab2-minecraft", minecraft_scan_response)

zgrab2.register_scan_response_type("minecraft", minecraft_scan_response)
