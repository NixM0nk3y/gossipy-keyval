#!/bin/bash
#
#
#

# data we pass in via environment
SDHOST=${SERVICE_DISCOVERY_HOST:-localhost}

# task properties
IP4_ADDRESS=$(curl --silent ${ECS_CONTAINER_METADATA_URI_V4} | jq -r .Networks[0].IPv4Addresses[0])
NODE_NAME=$(curl --silent ${ECS_CONTAINER_METADATA_URI_V4} | jq -r .DockerId)

exec /app/gossipy -clusterip ${IP4_ADDRESS} -clusternode ${NODE_NAME} -servicediscoveryhost ${SDHOST}