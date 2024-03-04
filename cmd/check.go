package cmd

import (
	"fmt"

	"github.com/jumppad-labs/jumppad/pkg/clients/system"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Checks the system to ensure required dependencies are installed",
	Long:  `Checks the system to ensure required dependencies are installed`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
		s := system.SystemImpl{}
		status := s.Preflight()

		fmt.Println(WhiteText.Render("Checking required system dependencies"))
		fmt.Println()

		gitStatus := GreenIcon.Render("✔")
		if !status.Git {
			gitStatus = RedIcon.Render("✘")
		}
		fmt.Println(gitStatus + WhiteText.Render(" Git"))

		containerStatus := GreenIcon.Render("✔")
		if !status.Docker && !status.Podman {
			containerStatus = RedIcon.Render("✘")
		}
		fmt.Println(containerStatus + WhiteText.Render(" Docker/Podman"))

		fmt.Println()
		for _, err := range status.Errors {
			fmt.Println(RedIcon.Render("ERROR") + WhiteText.Render(" "+err.Error()))
		}
		fmt.Println()
	},
}
