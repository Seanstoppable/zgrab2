#!/usr/bin/env python3
"""Validate Minecraft integration test output."""

import json
import os
import sys


def validate_minecraft_result(filepath, expect_raw_response=False):
    """Validate that a Minecraft scan result has the expected fields."""
    with open(filepath) as f:
        data = json.load(f)

    mc = data.get("data", {}).get("minecraft", {})
    status = mc.get("status")
    if status != "success":
        print(f"FAIL: Expected status 'success', got '{status}' in {filepath}")
        sys.exit(1)

    result = mc.get("result", {})

    # Validate version
    version = result.get("version", {})
    if not version.get("name"):
        print(f"FAIL: Missing version.name in {filepath}")
        sys.exit(1)
    if version.get("protocol") is None:
        print(f"FAIL: Missing version.protocol in {filepath}")
        sys.exit(1)
    # We configured the server as 1.20.1 (protocol 763)
    if "1.20.1" not in version["name"]:
        print(f"WARN: Expected version containing '1.20.1', got '{version['name']}'")
    if version["protocol"] != 763:
        print(f"WARN: Expected protocol 763, got {version['protocol']}")

    # Validate players
    players = result.get("players", {})
    if players.get("max") is None:
        print(f"FAIL: Missing players.max in {filepath}")
        sys.exit(1)
    if players.get("online") is None:
        print(f"FAIL: Missing players.online in {filepath}")
        sys.exit(1)

    # Validate description
    description = result.get("description")
    if not description:
        print(f"FAIL: Missing description in {filepath}")
        sys.exit(1)

    # Validate raw_response only if verbose
    if expect_raw_response and not result.get("raw_response"):
        print(f"FAIL: Expected raw_response in verbose output {filepath}")
        sys.exit(1)

    print(f"OK: {filepath}")
    print(f"    Version: {version['name']} (protocol {version['protocol']})")
    print(f"    Players: {players['online']}/{players['max']}")
    print(f"    Description: {description}")


if __name__ == "__main__":
    zgrab_output = os.path.join(
        os.popen("git rev-parse --show-toplevel").read().strip(),
        "zgrab-output",
        "minecraft",
    )

    validate_minecraft_result(
        os.path.join(zgrab_output, "minecraft.json"),
        expect_raw_response=False,
    )
    validate_minecraft_result(
        os.path.join(zgrab_output, "minecraft-verbose.json"),
        expect_raw_response=True,
    )

    print("All Minecraft integration tests passed!")
