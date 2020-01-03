// TODO setting env vars in go behaves like bash
// i.e you cannot set the current shells env vars from a program
// as the application runs in a child process
// need to look at better options

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Env struct {
	file *os.File
}

func NewEnv(envfile string) (*Env, error) {
	f, err := os.OpenFile(envfile, os.O_CREATE, 0655)
	if err != nil {
		return nil, err
	}

	return &Env{file: f}, nil
}

func (e *Env) Set(key, value string) error {
	// get the previous env var
	v := os.Getenv(key)
	if v != "" {
		_, err := e.file.WriteString(fmt.Sprintf(`%s=%s\n`, key, value))
		if err != nil {
			return err
		}
	}

	// set the new env var
	return os.Setenv(key, value)
}

// Clears all env vars restoring previous values
func (e *Env) Clear() error {
	e.file.Seek(0, 0)

	scanner := bufio.NewScanner(e.file)
	for scanner.Scan() {
		p := strings.Split(scanner.Text(), "=")
		os.Setenv(p[0], p[1])
	}

	return nil
}

func (e *Env) Close() {
	e.file.Close()
}
