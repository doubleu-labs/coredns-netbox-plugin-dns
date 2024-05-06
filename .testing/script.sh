#!/bin/bash

docker compose -p coredns-netbox-plugin-dns -f ./.testing/docker-compose.yml up -d && \

until [[ "`docker inspect -f {{.State.Health.Status}} coredns-netbox-plugin-dns-netbox-1`" == "healthy" ]]; do
    echo "Waiting for Netbox to come online..."
    sleep 5;
done && \

go run ./.testing/init/init.go
