package ui

import "github.com/charmbracelet/lipgloss"

// Color palette — green pitch, with accents.
var (
	colGreen  = lipgloss.Color("42")  // pitch green
	colDim    = lipgloss.Color("240") // muted text
	colSubtle = lipgloss.Color("245")
	colText   = lipgloss.Color("252")
	colLive   = lipgloss.Color("203") // red-ish for LIVE
	colWin    = lipgloss.Color("42")
	colAccent = lipgloss.Color("214") // amber
	colBg     = lipgloss.Color("236")
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("231")).
			Background(colGreen).
			Padding(0, 1)

	tabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("231")).
			Background(colGreen).
			Padding(0, 2)

	tabInactive = lipgloss.NewStyle().
			Foreground(colSubtle).
			Padding(0, 2)

	paneBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colDim)

	paneBorderActive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colGreen)

	paneTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colAccent)

	selectedRow = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("231")).
			Background(colBg)

	dimStyle  = lipgloss.NewStyle().Foreground(colDim)
	textStyle = lipgloss.NewStyle().Foreground(colText)

	liveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("231")).
			Background(colLive).
			Padding(0, 1)

	winStyle = lipgloss.NewStyle().Bold(true).Foreground(colWin)

	loseStyle = lipgloss.NewStyle().Foreground(colDim)

	champStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("231")).
			Background(colAccent).
			Padding(0, 1)

	liveDot = lipgloss.NewStyle().Bold(true).Foreground(colLive)

	bracketConn = lipgloss.NewStyle().Foreground(colDim)

	footerStyle = lipgloss.NewStyle().Foreground(colDim)

	statusOK  = lipgloss.NewStyle().Foreground(colGreen)
	statusErr = lipgloss.NewStyle().Foreground(colLive)

	headerKey = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
)
