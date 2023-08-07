package view

import tea "github.com/charmbracelet/bubbletea"

// messageWriter is a io.Writer that sends messages to a
// bubbletea view
type messageWriter struct {
	program *tea.Program
}

func (m *messageWriter) Write(b []byte) (int, error) {
	if m.program != nil {
		m.program.Send(LogMsg(b))
		return len(b), nil
	}

	return 0, nil
}
