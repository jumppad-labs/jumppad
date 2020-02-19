# Shipyard

![](https://github.com/shipyard-run/shipyard/workflows/Build/badge.svg)  
![](https://github.com/shipyard-run/shipyard/workflows/Release/badge.svg)  
  
[![codecov](https://codecov.io/gh/shipyard-run/shipyard/branch/master/graph/badge.svg)](https://codecov.io/gh/shipyard-run/shipyard)

![](./shipyard_horizontal.png)

## Contributing

We love contributions to the project, to contribute, first ensure that there is an issue and that it has been acknoledged by one of the maintainers of the project. Ensuring an issue exists and has been acknowledged ensures that the work you are about to submit will not be rejected due to specifications or duplicate work.
Once an issue exists, you can modify the code and raise a PR against this repo. We are working on increasing code coverage, please ensure that your work has at least 80% test coverage before submitting.


## Testing:

The project has two types of test, pure code Unit tests and, Functional tests which apply real blueprints to a locally running Docker engine and test output.  Currently only Unit tests are running in CI.

### Unit tests:

To run the unit tests you can use the make recipie `make test_unit` this runs the `go test` and excludes the functional tests.

```shell
shipyard on  master via 🐹 v1.13.5 on 🐳 v19.03.5 () 
➜ make test_unit                                                                  
go test -v -race github.com/shipyard-run/shipyard github.com/shipyard-run/shipyard/cmd github.com/shipyard-run/shipyard/pkg/clients github.com/shipyard-run/shipyard/pkg/clients/mocks github.com/shipyard-run/shipyard/pkg/config github.com/shipyard-run/shipyard/pkg/providers github.com/shipyard-run/shipyard/pkg/shipyard github.com/shipyard-run/shipyard/pkg/utils
testing: warning: no tests to run
PASS
ok      github.com/shipyard-run/shipyard        (cached) [no tests to run]
=== RUN   TestSetsEnvVar
--- PASS: TestSetsEnvVar (0.00s)
=== RUN   TestArgIsLocalRelativeFolder
--- PASS: TestArgIsLocalRelativeFolder (0.00s)
=== RUN   TestArgIsLocalAbsFolder
--- PASS: TestArgIsLocalAbsFolder (0.00s)
=== RUN   TestArgIsFolderNotExists
--- PASS: TestArgIsFolderNotExists (0.00s)
=== RUN   TestArgIsNotFolder
--- PASS: TestArgIsNotFolder (0.00s)
=== RUN   TestArgIsBlueprintFolder
--- PASS: TestArgIsBlueprintFolder (0.00s)
=== RUN   TestArgIsNotBlueprintFolder
```

### Functional tests:

To run the functional tests ensure that Docker is running in your environment then run `make test_functional` functional tests are executed with GoDog cucumber test runner for Go. Note: These tests create real blueprints and can a few minutes to run.

```shell
➜ make test_functional 
cd ./functional_tests && go test -v ./...
Feature: Docmentation
  In order to test the documentation feature
  something
  something

  Scenario: Documentation                                              # features/documentaion.feature:6
    Given the config "./test_fixtures/docs"                            # main_test.go:77 -> theConfig
2020-02-08T17:03:25.269Z [INFO]  Creating Network: ref=wan
2020-02-08T17:03:40.312Z [INFO]  Creating Documentation: ref=docs
2020-02-08T17:03:40.312Z [INFO]  Creating Container: ref=docs
2020-02-08T17:03:40.490Z [DEBUG] Attaching container to network: ref=docs network=wan
2020-02-08T17:03:41.187Z [INFO]  Creating Container: ref=terminal
2020-02-08T17:03:41.271Z [DEBUG] Attaching container to network: ref=terminal network=wan
    When I run apply                                                   # main_test.go:111 -> iRunApply
    Then there should be 1 network called "wan"                        # main_test.go:149 -> thereShouldBe1NetworkCalled
    And there should be 1 container running called "docs.wan.shipyard" # main_test.go:115 -> thereShouldBeContainerRunningCalled
    And a call to "http://localhost:8080/" should result in status 200 #
    
# ...

3 scenarios (3 passed)
16 steps (16 passed)
3m6.79622s
testing: warning: no tests to run
PASS
```

## Creating a release:

To create a release tag a commit `git tag <semver>` and push this to GitHub `git push origin <semver>` GitHub actions will build and create the release.
