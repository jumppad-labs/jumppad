git_commit = $(shell git log -1 --pretty=format:"%H")

test_unit:
	go test -v -race ./...

test_functional:
	go run main.go purge
	go run main.go test ./examples/container
	
	go run main.go purge
	go run main.go test ./examples/build

	go run main.go purge
	go run main.go test ./examples/docs

	go run main.go purge
	go run main.go test ./examples/nomad

	go run main.go purge
	go run main.go test ./examples/single_k3s_cluster
	
	go run main.go purge
	go run main.go test ./examples/multiple_k3s_clusters
	
	go run main.go purge
	go run main.go test ./examples/exec
	
	go run main.go purge
	go run main.go test ./examples/certificates
	
	go run main.go purge
	go run main.go test ./examples/terraform
	
	go run main.go purge
	go run main.go test ./examples/registiries

test_e2e_cmd: install_local
	jumppad up --no-browser ./examples/single_k3s_cluster
	jumppad down

test_docker:
	docker build -t shipyard-run/tests -f Dockerfile.test .
	docker run --rm shipyard-run/tests bash -c 'go test -v -race ./...'

test: test_unit test_functional

build: build-darwin build-linux build-windows

build-darwin:
	CGO_ENABLED=0 GOOS=darwin go build -ldflags "-X main.version=${git_commit}" -o bin/jumppad-darwin main.go

build-linux:
	CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.version=${git_commit}" -o bin/jumppad-linux main.go

build-linux-small:
	CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.version=${git_commit} -s -w" -o bin/jumppad-linux-small main.go

build-windows:
	CGO_ENABLED=0 GOOS=windows go build -ldflags "-X main.version=${git_commit}" -o bin/jumppad-windows.exe main.go

install_local:
	go build -ldflags "-X main.version=${git_commit}" -o bin/jumppad main.go
	sudo mv /usr/local/bin/jumppad /usr/local/bin/jumppad-old || true
	sudo cp bin/jumppad /usr/local/bin/jumppad

remove_local:
	sudo rm /usr/local/bin/jumppad
	sudo mv /usr/local/bin/jumppad-old /usr/local/bin/jumppad

test_releaser:
	goreleaser release --rm-dist --snapshot

generate_mocks:
	go install github.com/vektra/mockery/v2@v2.20.0
	go generate ./...
