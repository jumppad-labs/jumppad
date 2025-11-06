// This is a minimal test stub that stays alive for connector tests.
// It does NOT implement actual connector functionality - tests use mocks for that.
// It only needs to accept the right flags and stay running.
package main

import (
	"flag"
	"os"
	"time"
)

func main() {
	// Accept flags that connector tests pass
	flag.Bool("non-interactive", false, "non-interactive mode")
	flag.String("grpc-bind", "", "grpc bind address")
	flag.String("http-bind", "", "http bind address")
	flag.String("api-bind", "", "api bind address")
	flag.String("root-cert-path", "", "root cert path")
	flag.String("server-cert-path", "", "server cert path")
	flag.String("server-key-path", "", "server key path")
	flag.String("log-level", "", "log level")
	flag.Parse()

	// Verify subcommand is "connector run"
	args := flag.Args()
	if len(args) < 2 || args[0] != "connector" || args[1] != "run" {
		os.Exit(1)
	}

	// Stay alive so tests can verify IsRunning() returns true
	// Tests will kill us via the PID file
	time.Sleep(30 * time.Second)
}
