test_unit:
	go test -v -race $(shell go list ./... | grep -v /functional_tests)

test_functional:
	cd ./functional_tests && go test -v ./...

test: test_unit test_functional