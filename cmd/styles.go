package cmd

import "github.com/charmbracelet/lipgloss"

var HeaderText = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
var WhiteText = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
var GrayText = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

var YellowIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).PaddingRight(1)
var GrayIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).PaddingRight(1)
var RedIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).PaddingRight(1)
var GreenIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).PaddingRight(1)

var InfoLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).PaddingRight(1)
var DebugLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).PaddingRight(1)
var WarningLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).PaddingRight(1)
var ErrorLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).PaddingRight(1)
var SuccessLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).PaddingRight(1)
