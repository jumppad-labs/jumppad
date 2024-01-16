package changelog

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/timer"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var headerStyle = lipgloss.NewStyle().Bold(true)
var footerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("37"))

type model struct {
	seen     bool
	content  string
	timer    timer.Model
	viewport viewport.Model
	ready    bool
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return m.timer.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {

	// handle the timer tick
	case timer.TickMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return m, cmd

	case timer.TimeoutMsg:
		return m, tea.Quit

	case timer.StartStopMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return m, cmd

	// Is it a key press?
	case tea.KeyMsg:
		switch msg.String() {
		case "y":
			m.seen = true
			return m, tea.Quit
		}

	// if the mouse scrolls stop the timer as the user is reading
	case tea.MouseMsg:
		tCmd := m.timer.Stop()
		m.viewport, cmd = m.viewport.Update(msg)

		return m, tea.Batch(tCmd, cmd)

	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight

			// build the content with glamour
			renderer, err := glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(m.viewport.Width),
			)

			if err != nil {
				return m, tea.Quit
			}

			str, err := renderer.Render(m.content)
			if err != nil {
				return m, tea.Quit
			}

			m.viewport.SetContent(str)
			m.ready = true

			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the viewport one line below the header.
			m.viewport.YPosition = headerHeight + 1
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
}

func (m model) headerView() string {
	title := headerStyle.Render("Changelog")
	return lipgloss.JoinVertical(lipgloss.Top, title, "")
}

func (m model) footerView() string {
	keys := footerStyle.Render("press [y] to acknowledge changelog")
	info := footerStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	spacing := strings.Repeat(" ", max(0, m.viewport.Width-lipgloss.Width(keys)-lipgloss.Width(info)))

	line := lipgloss.JoinHorizontal(lipgloss.Left, keys, spacing, info)

	return lipgloss.JoinVertical(lipgloss.Top, "", line)
}

func initalModel(content string) model {
	return model{seen: false, content: content, timer: timer.NewWithInterval(15*time.Second, 100*time.Millisecond)}
}
