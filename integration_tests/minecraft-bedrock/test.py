#!/usr/bin/env python3
"""Validate Minecraft Bedrock integration test output."""

import json
import os
import sys


def validate_bedrock_result(filepath, expect_raw_response=False):
    """Validate that a Minecraft Bedrock scan result has the expected fields."""
    with open(filepath) as f:
        data = json.load(f)

    mc = data.get("data", {}).get("minecraft-bedrock", {})
    status = mc.get("status")
    if status != "success":
        print(f"FAIL: Expected status 'success', got '{status}' in {filepath}")
        sys.exit(1)

    result = mc.get("result", {})

    # Validate edition
    edition = result.get("edition", "")
    if edition not in ("MCPE", "MCEE"):
        print(f"FAIL: Unexpected edition '{edition}' in {filepath}")
        sys.exit(1)

    # Validate version
    version = result.get("version")
    if not version:
        print(f"FAIL: Missing version in {filepath}")
        sys.exit(1)

    # Validate MOTD
    motd = result.get("motd")
    if not motd:
        print(f"FAIL: Missing motd in {filepath}")
        sys.exit(1)

    # Validate players
    if result.get("max_players") is None:
        print(f"FAIL: Missing max_players in {filepath}")
        sys.exit(1)

    # Validate raw_response only if verbose
    if expect_raw_response and not result.get("raw_response"):
        print(f"FAIL: Expected raw_response in verbose output {filepath}")
        sys.exit(1)

    print(f"OK: {filepath}")
    print(f"    Edition: {edition}")
    print(f"    Version: {version} (protocol {result.get('protocol')})")
    print(f"    Players: {result.get('online_players')}/{result.get('max_players')}")
    print(f"    MOTD: {motd}")
    if result.get("game_mode"):
        print(f"    GameMode: {result.get('game_mode')}")


if __name__ == "__main__":
    zgrab_output = os.path.join(
        os.popen("git rev-parse --show-toplevel").read().strip(),
        "zgrab-output",
        "minecraft-bedrock",
    )

    validate_bedrock_result(
        os.path.join(zgrab_output, "minecraft-bedrock.json"),
        expect_raw_response=False,
    )
    validate_bedrock_result(
        os.path.join(zgrab_output, "minecraft-bedrock-verbose.json"),
        expect_raw_response=True,
    )

    print("All Minecraft Bedrock integration tests passed!")
