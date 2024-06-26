FROM golang:1.22.4-alpine3.19 AS spicedb-builder


# Create and change to the app directory.
WORKDIR /app

# Retrieve application dependencies.
# This allows the container build to reuse cached dependencies.
# Expecting to copy go.mod and if present go.sum.
COPY go.* ./
RUN go mod download

# Copy local code to the container image.
COPY . ./

# Build the binary.
RUN go build -v -o spicedb ./cmd/spicedb/

# Use the official Debian slim image for a lean production container.
# https://hub.docker.com/_/debian
# https://docs.docker.com/develop/develop-images/multistage-build/#use-multi-stage-builds
FROM debian:buster-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates &&  \
    apt install dumb-init && \
    rm -rf /var/lib/apt/lists/*

# Copy the binary to the production image from the builder stage.
COPY --from=spicedb-builder /app/spicedb /app/spicedb
# Run the web service on container startup.
# migrate up --database-engine postgres --database-uri postgres://postgres:postgres@%s/cog-analytics-backend:us-central1:permify/postgres
# or RUN apt install tini
# COPY --from=ghcr.io/grpc-ecosystem/grpc-health-probe:v0.4.12 /ko-app/grpc-health-probe /usr/local/bin/grpc_health_probe

ARG PRESHAREDKEY


ENV PATH="$PATH:/usr/local/bin"
EXPOSE 50051
# Copy the entrypoint script
COPY entrypoint.sh /entrypoint.sh

# Make the entrypoint script executable
RUN chmod +x /entrypoint.sh

# Use the entrypoint script
ENTRYPOINT ["/entrypoint.sh"]