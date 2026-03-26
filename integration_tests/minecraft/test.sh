#!/usr/bin/env bash

set -e
MODULE_DIR=$(dirname $0)
ZGRAB_ROOT=$(git rev-parse --show-toplevel)
ZGRAB_OUTPUT=$ZGRAB_ROOT/zgrab-output

mkdir -p $ZGRAB_OUTPUT/minecraft

CONTAINER_NAME=zgrab_minecraft

# The Minecraft server can take a while to start up. Wait for it to be ready
# by checking if port 25565 is accepting connections.
echo "minecraft/test: Waiting for Minecraft server to be ready..."
MAX_WAIT=120
WAITED=0
while ! docker exec $CONTAINER_NAME bash -c 'echo | nc -w1 localhost 25565' > /dev/null 2>&1; do
    if [ $WAITED -ge $MAX_WAIT ]; then
        echo "minecraft/test: Timed out waiting for server after ${MAX_WAIT}s"
        docker logs --tail 50 $CONTAINER_NAME
        exit 1
    fi
    sleep 5
    WAITED=$((WAITED + 5))
    echo "minecraft/test: Still waiting... (${WAITED}s)"
done
echo "minecraft/test: Server is ready after ${WAITED}s"

OUTPUT_FILE=$ZGRAB_OUTPUT/minecraft/minecraft.json

echo "minecraft/test: Running zgrab2 minecraft scan"
CONTAINER_NAME=$CONTAINER_NAME $ZGRAB_ROOT/docker-runner/docker-run.sh minecraft --port 25565 > $OUTPUT_FILE

# Also test with verbose flag
OUTPUT_FILE_VERBOSE=$ZGRAB_OUTPUT/minecraft/minecraft-verbose.json
echo "minecraft/test: Running zgrab2 minecraft scan (verbose)"
CONTAINER_NAME=$CONTAINER_NAME $ZGRAB_ROOT/docker-runner/docker-run.sh minecraft --port 25565 --verbose > $OUTPUT_FILE_VERBOSE

# Dump the docker logs
echo "minecraft/test: BEGIN docker logs from $CONTAINER_NAME [{("
docker logs --tail 30 $CONTAINER_NAME
echo ")}] END docker logs from $CONTAINER_NAME"
