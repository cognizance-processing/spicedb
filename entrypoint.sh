#!/bin/sh

# Access the environment variable
PRESHAREDKEY=${PRESHAREDKEY}
DBUSER=${DBUSER}
DBPASSWORD=${DBPASSWORD}
DBIP=${DBIP}

# Replace the placeholder in your command with the environment variable
exec /app/spicedb migrate head \
    --grpc-preshared-key "${PRESHAREDKEY}" \
    --datastore-engine=postgres \
    --datastore-conn-uri="postgres://${DBUSER}:${DBPASSWORD}@${DBIP}:5432/spicedb?sslmode=disable"
