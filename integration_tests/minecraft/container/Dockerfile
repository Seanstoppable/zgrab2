FROM itzg/minecraft-server:java21

# Accept the Minecraft EULA
ENV EULA=TRUE
# Use a specific version for reproducible tests
ENV VERSION=1.20.1
# Offline mode so no Mojang auth is needed
ENV ONLINE_MODE=FALSE
# Set a known MOTD for test validation
ENV MOTD=zgrab2-integration-test
# Reduce resource usage for CI
ENV MEMORY=512M
ENV MAX_PLAYERS=20

EXPOSE 25565
