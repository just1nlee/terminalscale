package main

import "charm.land/lipgloss/v2"

const (
	PanePaddingH    = 1 // left/right padding inside border
	PanePaddingV    = 0 // top/bottom padding inside border
	PaneMarginH     = 0 // left/right margin outside border
	PaneMarginV     = 0 // top/bottom margin outside border
	StatusBarheight = 1
)

var (
	colorBackground  = lipgloss.Color("#0d0d0d")
	colorGreenBright = lipgloss.Color("#00e5a0")
	colorGreenDim    = lipgloss.Color("#1a6b4a")
	colorWhite       = lipgloss.Color("#ebebeb")

	focusedPaneStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colorGreenDim).
				Padding(PanePaddingV, PanePaddingH).
				Margin(PaneMarginV, PaneMarginH)

	unfocusedPaneStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colorWhite).
				Padding(PanePaddingV, PanePaddingH).
				Margin(PaneMarginV, PaneMarginH)
)
