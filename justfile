
_compose-common := "--project-directory='.testing' --file=.testing/docker-compose.yml"

_ensure-podman-sock:
    systemctl --user start podman.socket

instance-start: _ensure-podman-sock
    podman compose {{_compose-common}} up --build --detach
    until [[ "`podman inspect -f {{{{.State.Health.Status}} testing-netbox-1`" == "healthy" ]]; do \
        echo "Waiting for Netbox to come online..."; \
        sleep 5; \
    done
    go run .testing/init/init.go

instance-stop: _ensure-podman-sock
    podman compose {{_compose-common}} down --volumes

test: instance-start
    go test \
        -coverprofile=coverage.out \
        -coverpkg=github.com/doubleu-labs/coredns-netbox-plugin-dns,github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox \
        .

coverage: test
    go tool cover -html=coverage.out
