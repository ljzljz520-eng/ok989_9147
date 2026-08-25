# ok989_9147

Go project for module `internalauth`.

## Standard commands

```bash
go build ./...
go test -count=1 ./...
```

## Run

```bash
go run .
go run ./cmd/server
```

## Frontend

```bash
(cd web && npm run build)
```

## Docker validation

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh my-go-task linux/arm64
./build_benzhi_docker.sh my-go-task linux/amd64
docker run -it my-go-task:latest
```

## Known initial failures

Initial validation failures are retained in the package command output and run logs; they are not copied into the project repository.
