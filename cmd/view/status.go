package view

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var statusSpinnerDuration = 100 * time.Millisecond

// StatusMsg updates the status bar, optionally the elapsed time and
// spinner can be enabled
type StatusMsg struct {
	Message     string
	ShowElapsed bool
}

type StatusTickMsg struct {
	spinnerMsg tea.Msg
}

type StatusModel struct {
	spinner     spinner.Model
	message     string
	showSpinner bool
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
	case StatusTickMsg:
		m.spinner, _ = m.spinner.Update(msg.spinnerMsg)

	case StatusMsg:
		m.message = msg.Message
		m.showSpinner = msg.ShowElapsed
	}

	// after updating the spinner or after the spinner has been enabled
	// by a status message
	// we need to return a command to ensure that the spinner is updated again after an interval
	if m.showSpinner {
		return m, m.tick()
	}

	return m, nil
}

func (m StatusModel) View() string {
	if m.showSpinner {
		spinner := lipgloss.NewStyle().MarginRight(1).Render(m.spinner.View())
		return lipgloss.JoinHorizontal(lipgloss.Left, spinner, m.message)
	}

	message := lipgloss.NewStyle().MarginLeft(1).Render(m.message)
	return lipgloss.JoinHorizontal(lipgloss.Left, message)
}

// tick ensures that the spinner is updated
func (m StatusModel) tick() tea.Cmd {
	return tea.Tick(statusSpinnerDuration, func(t time.Time) tea.Msg {
		return StatusTickMsg{m.spinner.Tick()}
	})
}
