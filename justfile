
_compose-common := "--project-directory='.testing' --file=.testing/docker-compose.yml"

_ensure-podman-sock:
    systemctl --user start podman.socket

[script("bash")]
instance-start: _ensure-podman-sock
    podman compose {{_compose-common}} up --build --detach
    until [[ "`podman inspect -f {{{{.State.Health.Status}} testing-netbox-1`" == "healthy" ]]; do \
        echo "Waiting for Netbox to come online..."; \
        sleep 5; \
    done
    go run .testing/init.go

instance-stop: _ensure-podman-sock
    podman compose {{_compose-common}} down --volumes

test: instance-start
    rm coverage.out
    go clean -testcache
    go test \
        -covermode=atomic \
        -coverpkg=./... \
        -coverprofile=coverage.out \
        ./... \
        -run ./...

coverage: test
    go tool cover -html=coverage.out
