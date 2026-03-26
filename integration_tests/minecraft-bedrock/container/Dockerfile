FROM itzg/minecraft-bedrock-server

# Accept the Minecraft EULA
ENV EULA=TRUE
# Use a specific version for reproducible tests
ENV VERSION=1.20.1.02
# Set a known server name for test validation
ENV SERVER_NAME=zgrab2-bedrock-test
# Reduce resource usage for CI
ENV MAX_PLAYERS=20

EXPOSE 19132/udp
