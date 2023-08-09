package view

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/muesli/reflow/wordwrap"
)

var tickDuration = 250 * time.Millisecond

// LogMsg sends data to the log panel
type LogMsg string
type ErrMsg error
type TickMsg time.Time

type model struct {
	height int
	width  int
	left   int
	top    int

	viewport  viewport.Model
	statusbar StatusModel
	messages  []string
	follow    bool
	logger    logger.Logger
}

type KeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Quit   key.Binding
	Filter key.Binding
	Level  key.Binding
}

var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("↓/j", "move down"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Filter: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "filter logs"),
	),
	Level: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "log level"),
	),
}

func initialModel() model {
	status := NewStatus()

	return model{
		messages:  []string{},
		statusbar: status,
		follow:    true,
		left:      1,
	}
}

func (m model) Init() tea.Cmd {
	// init child models
	return tea.Batch(m.statusbar.Init(), m.tick())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// when receiving a tick all components that require
	// tick updates must be called
	// tick also must be called to ensure that the tick
	// continues
	case TickMsg:
		// update the status model so that the spinner updates
		sm, sCmd := m.statusbar.Update(msg)
		m.statusbar = sm

		return m, tea.Batch(sCmd, m.tick())

	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseWheelUp:
			m.follow = false
		case tea.MouseWheelDown:
			// noop for now
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.Up):
			m.follow = false
		case key.Matches(msg, DefaultKeyMap.Down):
			// noop
		case key.Matches(msg, DefaultKeyMap.Level):
			if m.logger.IsDebug() {
				m.logger.SetLevel(logger.LogLevelInfo)
			} else {
				m.logger.SetLevel(logger.LogLevelDebug)
			}

		case key.Matches(msg, DefaultKeyMap.Quit):
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		// this message is always fired when first displaying the view
		// then after every terminal resize
		headerHeight := 2 //lipgloss.Height(m.headerView())
		footerHeight := 2 //lipgloss.Height(m.footerView())

		m.height = msg.Height
		m.width = msg.Width

		m.viewport = viewport.New(m.width-m.left, m.height-headerHeight-footerHeight-m.top)
		return m, nil

	case LogMsg:
		// wrap the current message over multiple lines to ensure that it
		// does not go beyond the bounds of the terminal
		message := wrapMessage(string(msg), m.viewport.Width)

		// append the new log lines to the current buffer
		m.messages = appendMessage(m.messages, message)

		// set the content on the viewport
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		if m.follow {
			m.viewport.GotoBottom()
		}

		// render the log lines
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)

		return m, cmd

	case StatusMsg:
		var cmd tea.Cmd
		m.statusbar, cmd = m.statusbar.Update(msg)

		return m, cmd

	// we handle errors just like any other message
	case ErrMsg:
		//m.err = msg

		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	// if there is a 0 width do not draw
	if m.width-m.left < 1 {
		return ""
	}

	return lipgloss.JoinVertical(lipgloss.Top,
		lipgloss.NewStyle().MarginLeft(m.left).Render(m.headerView()),
		lipgloss.NewStyle().MarginLeft(m.left).Render(m.viewportView()),
		lipgloss.NewStyle().MarginLeft(m.left).Render(m.footerView()),
	)
}

func (m model) viewportView() string {
	return m.viewport.View()
}

func (m model) headerView() string {
	title := "Jumppad Dev Mode"
	line := lipgloss.NewStyle().Foreground(lipgloss.Color("37")).Render(strings.Repeat("─", m.width-m.left))
	return lipgloss.JoinVertical(lipgloss.Top, title, line)
}

func (m model) footerView() string {
	line := lipgloss.NewStyle().Foreground(lipgloss.Color("37")).Render(strings.Repeat("─", m.width-m.left))

	return lipgloss.JoinVertical(lipgloss.Top, line, m.statusbar.View())
}

// tick ensures that there are regular heartbeats for components that need them
func (m model) tick() tea.Cmd {
	return tea.Tick(tickDuration, func(t time.Time) tea.Msg {
		return TickMsg(time.Now())
	})
}

func wrapMessage(m string, width int) string {
	message := wordwrap.String(m, width)
	return strings.TrimSuffix(message, "\n")
}

func appendMessage(messages []string, message string) []string {
	lines := strings.Split(message, "\n")
	for i, l := range lines {
		if i > 0 {
			lines[i] = fmt.Sprintf("     %s", l)
		}
	}

	message = strings.Join(lines, "\n")
	messages = append(messages, message)

	// if greater than 10000 pop first
	if len(messages) > 10000 {
		messages = messages[1:]
	}

	return messages
}
