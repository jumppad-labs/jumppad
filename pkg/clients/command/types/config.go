package types

import "time"

type CommandConfig struct {
	Command          string
	Args             []string
	Env              []string
	WorkingDirectory string
	RunInBackground  bool
	LogFilePath      string
	Timeout          time.Duration
}
