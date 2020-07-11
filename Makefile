git_commit = $(shell git log -1 --pretty=format:"%H")

test_unit:
	go clean --cache
	go test -v -race ./...

test_functional:
	go run main.go purge
	go run main.go test ./examples/container

	go run main.go purge
	go run main.go test ./examples/docs

	go run main.go purge
	go run main.go test ./examples/modules
	
	go run main.go purge
	go run main.go test ./examples/nomad

	go run main.go purge
	go run main.go test ./examples/single_k3s_cluster

test_docker:
	docker build -t shipyard-run/tests -f Dockerfile.test .
	docker run --rm shipyard-run/tests bash -c 'go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...'
	docker run --rm shipyard-run/tests bash -c 'go test -v ./pkg/shipyard'

test: test_unit test_functional

build: build-darwin build-linux build-windows

build-darwin:
	CGO_ENABLED=0 GOOS=darwin go build -ldflags "-X main.version=${git_commit}" -o bin/yard-darwin main.go

build-linux:
	CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.version=${git_commit}" -o bin/yard-linux main.go

build-windows:
	CGO_ENABLED=0 GOOS=windows go build -ldflags "-X main.version=${git_commit}" -o bin/yard-windows.exe main.go

install_local:
	go build -ldflags "-X main.version=${git_commit}" -o bin/yard-dev main.go
	sudo cp bin/yard-dev /usr/local/bin/yard-dev
