#!/bin/sh

# Access the environment variable
PRESHAREDKEY=${PRESHAREDKEY}

# Replace the placeholder in your command with the environment variable
exec /app/spicedb serve \
    --grpc-preshared-key "${PRESHAREDKEY}" \
    --datastore-engine=postgres \
    --datastore-conn-uri="postgres://new:Happy456@34.44.110.10:5432/spicedb?sslmode=disable"
