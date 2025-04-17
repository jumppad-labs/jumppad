package changelog

import (
	"os"
	"path"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/instruqt/jumppad/pkg/utils"
	"github.com/mattn/go-isatty"
)

type Changelog struct {
}

// Show is a blocking function that displays the changelog
func (c *Changelog) Show(content, version string, alwaysShow bool) error {
	// if not a terminal do not show the changelog
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		return nil
	}

	changelog := path.Join(utils.JumppadHome(), "changelog")
	changeFile := path.Join(changelog, version)

	// check if the change log has been acknowledged, if so do not show
	_, err := os.Stat(changeFile)
	if err == nil && !alwaysShow {
		return nil
	}

	p := tea.NewProgram(initalModel(content), tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		return err
	}

	// write a file to the .jumppad folder that contains the version
	os.MkdirAll(changelog, os.ModePerm)
	os.WriteFile(changeFile, []byte(version), os.ModePerm)

	return nil
}
