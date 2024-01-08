package cmd

import "github.com/charmbracelet/lipgloss"

var headerText = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
var whiteText = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
var grayText = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

var yellowIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).PaddingRight(1)
var grayIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).PaddingRight(1)
var redIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).PaddingRight(1)
var greenIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).PaddingRight(1)

var infoLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).PaddingRight(1)
var debugLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).PaddingRight(1)
var warningLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).PaddingRight(1)
var errorLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).PaddingRight(1)
var successLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).PaddingRight(1)
