release:
	goreleaser release --snapshot --clean

.PHONY: front-sync build-main-server build-kshield
front-sync:
	powershell -NoProfile -Command "if (Test-Path cmd/main-server/dist) { Remove-Item -Recurse -Force cmd/main-server/dist }; Copy-Item -Recurse front cmd/main-server/dist"

build-main-server: front-sync
	CGO_ENABLED=0 go build -o main-server.exe ./cmd/main-server

build-kshield:
	CGO_ENABLED=0 go build -o kshield.exe ./cmd/kshield

build:
	go build -o ./ciscollector ./cmd/ciscollector
run: 
	go build -o ./ciscollector ./cmd/ciscollector && ./ciscollector -r

run-manual-json:
	go build -o ./ciscollector ./cmd/ciscollector && ./ciscollector -config . -r --json

run-cron-json:
	go build -o ./ciscollector ./cmd/ciscollector && ./ciscollector --setup-cron -config . --json
linux: 
	GOOS=linux GOARCH=amd64 go build -o ./linux/ciscollector ./cmd/ciscollector

install:
	go install ./cmd/ciscollector
	go install ./docker_testing/integrationtest