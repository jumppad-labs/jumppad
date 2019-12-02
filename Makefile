test_unit:
	go test -v -race $(shell go list ./... | grep -v /functional_tests)

test_functional:
	cd ./functional_tests && go test -v ./...

test: test_unit test_functional

# Run tests continually with  a watcher
autotest:
	filewatcher --idle-timeout 24h -x **/functional_tests gotestsum