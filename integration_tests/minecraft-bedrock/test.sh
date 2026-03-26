#!/usr/bin/env bash

set -e
MODULE_DIR=$(dirname $0)
ZGRAB_ROOT=$(git rev-parse --show-toplevel)
ZGRAB_OUTPUT=$ZGRAB_ROOT/zgrab-output

mkdir -p $ZGRAB_OUTPUT/minecraft-bedrock

CONTAINER_NAME=zgrab_minecraft-bedrock

# The Bedrock server can take a while to start up. Wait for it to be ready.
echo "minecraft-bedrock/test: Waiting for Bedrock server to be ready..."
MAX_WAIT=120
WAITED=0
while ! docker exec $CONTAINER_NAME bash -c 'echo | nc -u -w1 localhost 19132' > /dev/null 2>&1; do
    if [ $WAITED -ge $MAX_WAIT ]; then
        echo "minecraft-bedrock/test: Timed out waiting for server after ${MAX_WAIT}s"
        docker logs --tail 50 $CONTAINER_NAME
        exit 1
    fi
    sleep 5
    WAITED=$((WAITED + 5))
    echo "minecraft-bedrock/test: Still waiting... (${WAITED}s)"
done
echo "minecraft-bedrock/test: Server is ready after ${WAITED}s"

OUTPUT_FILE=$ZGRAB_OUTPUT/minecraft-bedrock/minecraft-bedrock.json

echo "minecraft-bedrock/test: Running zgrab2 minecraft-bedrock scan"
CONTAINER_NAME=$CONTAINER_NAME $ZGRAB_ROOT/docker-runner/docker-run.sh minecraft-bedrock --port 19132 > $OUTPUT_FILE

# Also test with verbose flag
OUTPUT_FILE_VERBOSE=$ZGRAB_OUTPUT/minecraft-bedrock/minecraft-bedrock-verbose.json
echo "minecraft-bedrock/test: Running zgrab2 minecraft-bedrock scan (verbose)"
CONTAINER_NAME=$CONTAINER_NAME $ZGRAB_ROOT/docker-runner/docker-run.sh minecraft-bedrock --port 19132 --verbose > $OUTPUT_FILE_VERBOSE

# Dump the docker logs
echo "minecraft-bedrock/test: BEGIN docker logs from $CONTAINER_NAME [{("
docker logs --tail 30 $CONTAINER_NAME
echo ")}] END docker logs from $CONTAINER_NAME"
