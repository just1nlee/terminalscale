package main

import "charm.land/lipgloss/v2"

const (
	PanePaddingH    = 1 // left/right padding inside border
	PanePaddingV    = 0 // top/bottom padding inside border
	PaneMarginH     = 0 // left/right margin outside border
	PaneMarginV     = 0 // top/bottom margin outside border
	CursorOffsetX   = 1 + PanePaddingH + PaneMarginH
	CursorOffsetY   = 1 + PanePaddingV + PaneMarginV
	StatusBarHeight = 1
)

var (
	colorBackground  = lipgloss.Color("#0d0d0d")
	colorYellow      = lipgloss.Color("#e5cb30")
	colorLightYellow = lipgloss.Color("#edd16d")
	colorLightGray   = lipgloss.Color("#b6b7bb")
	colorDarkGray    = lipgloss.Color("#2d2d2f")
	colorWhite       = lipgloss.Color("#ebebeb")

	focusedPaneStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colorLightYellow).
				Padding(PanePaddingV, PanePaddingH).
				Margin(PaneMarginV, PaneMarginH)

	unfocusedPaneStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colorLightGray).
				Padding(PanePaddingV, PanePaddingH).
				Margin(PaneMarginV, PaneMarginH)

	statusBarStyle = lipgloss.NewStyle().
			Background(colorBackground).
			Foreground(colorWhite)
)

func paneExtraW() int {
	return 2 + (PanePaddingH * 2) + (PaneMarginH * 2)
}

func paneExtraH() int {
	return 2 + (PanePaddingV * 2) + (PaneMarginV * 2)
}
