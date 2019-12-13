package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"

	"context"

	getter "github.com/hashicorp/go-getter"
	"github.com/shipyard-run/cli/pkg/shipyard"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply [file] [directory] ...",
	Short: "Apply the supplied stack configuration",
	Long:  `Apply the supplied stack configuration`,
	Example: `  # Recursively create a stack from a directory
  yard apply my-stack

  # Create a stack from a specific file
  yard apply my-stack/network.hcl
	`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {

		pwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("Error getting wd: %s", err)
		}

		dst := fmt.Sprintf("%s/.shipyard/blueprints/consul-k8s", os.Getenv("HOME"))

		// if the argument is a url fetch it first
		c := &getter.Client{
			Ctx:     context.Background(),
			Src:     "github.com/shipyard-run/blueprints//consul-k8s",
			Dst:     dst,
			Pwd:     pwd,
			Mode:    getter.ClientModeAny,
			Options: []getter.ClientOption{},
		}

		err = c.Get()
		if err != nil {
			panic(err)
		}

		// Do Stuff Here
		e, err := shipyard.NewWithFolder(dst)
		if err != nil {
			panic(err)
		}

		// get the blueprint and open browsers
		fmt.Println("Title", e.Blueprint().Title)
		fmt.Println("Author", e.Blueprint().Author)
		fmt.Println("")
		fmt.Println(e.Blueprint().Intro)
		fmt.Println("")

		err = e.Apply()
		if err != nil {
			panic(err)
		}

		openCommand := "open"
		if runtime.GOOS == "linux" {
			openCommand = "xdg-open"
		}

		for _, b := range e.Blueprint().BrowserTabs {
			cmd := exec.Command(openCommand, b)
			cmd.Run()
		}
	},
}
