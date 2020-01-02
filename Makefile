test_unit:
	go test -v -race $(shell go list ./... | grep -v /functional_tests)

test_functional:
	cd ./functional_tests && go test -v ./...

test: test_unit test_functional

# Run tests continually with  a watcher
autotest:
	filewatcher --idle-timeout 24h -x **/functional_tests gotestsum --format standard-verbose

build: build-darwin build-linux build-windows

build-darwin:
	CGO_ENABLED=0 GOOS=darwin go build -o bin/yard-darwin main.go

build-linux:
	CGO_ENABLED=0 GOOS=linux go build -o bin/yard-linux main.go

build-windows:
	CGO_ENABLED=0 GOOS=windows go build -o bin/yard-windows.exe main.go

install_local:
	go build -o /usr/local/bin/yard-dev main.go