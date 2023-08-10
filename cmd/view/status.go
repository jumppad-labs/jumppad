package view

import (
	"fmt"
	"math"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StatusMsg updates the status bar, optionally the elapsed time and
// spinner can be enabled
type StatusMsg struct {
	Message     string
	ShowElapsed bool
}

type StatusModel struct {
	spinner     spinner.Model
	message     string
	showSpinner bool
	startTime   time.Time
	logLevel    string
}

func NewStatus() StatusModel {
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("36"))

	return StatusModel{
		spinner: sp,
		message: " ",
	}
}

func (m StatusModel) Init() tea.Cmd {
	return nil
}

func (m StatusModel) Update(msg tea.Msg) (StatusModel, tea.Cmd) {
	switch msg := msg.(type) {
	case TickMsg:
		// advance the spinner
		var spCmd tea.Cmd
		spMsg := m.spinner.Tick()
		m.spinner, spCmd = m.spinner.Update(spMsg)

		return m, spCmd

	case StatusMsg:
		m.message = msg.Message
		m.showSpinner = msg.ShowElapsed

		// if we are getting a new message that sets a timer
		// set the current start time to now so that it is possible to calculate
		// elapsed time
		if m.showSpinner {
			m.startTime = time.Now()
		}
	}

	return m, nil
}

func (m StatusModel) View() string {
	if m.showSpinner {
		et := time.Since(m.startTime)
		elapsed := fmt.Sprintf("%ds", int(math.Round(float64(et)/float64(time.Second))))

		spinner := lipgloss.NewStyle().MarginRight(1).Render(m.spinner.View())
		text := lipgloss.NewStyle().MarginRight(4).Render(m.message)

		return lipgloss.JoinHorizontal(lipgloss.Left, spinner, text, elapsed)
	}

	message := lipgloss.NewStyle().MarginLeft(1).Render(m.message)
	return lipgloss.JoinHorizontal(lipgloss.Left, message)
}
