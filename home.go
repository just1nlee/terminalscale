package main

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

var asciiArt = []string{
	` _____ _____ ____  __  __ ___ _   _    _    _     ____   ____    _    _     _____ `,
	`|_   _| ____|  _ \|  \/  |_ _| \ | |  / \  | |   / ___| / ___|  / \  | |   | ____|`,
	`  | | |  _| | |_) | |\/| || ||  \| | / _ \ | |   \___ \| |     / _ \ | |   |  _|  `,
	`  | | | |___|  _ <| |  | || || |\  |/ ___ \| |___ ___) | |___ / ___ \| |___| |___ `,
	`  |_| |_____|_| \_\_|  |_|___|_| \_/_/   \_\_____|____/ \____/_/   \_\_____|_____|`,
}

var hints = []string{
	"PRESS ? FOR HELP",
	"CTRL+Q TO EXIT",
}

func (m model) renderHomeScreen() string {
	artStyle := lipgloss.NewStyle().
		Foreground(colorYellow).
		Bold(true)

	hintStyle := lipgloss.NewStyle().
		Foreground(colorWhite)

	// Center each art line
	var lines []string
	for _, line := range asciiArt {
		padLen := (m.width - len(line)) / 2
		if padLen < 0 {
			padLen = 0
		}
		lines = append(lines, strings.Repeat(" ", padLen)+artStyle.Render(line))
	}

	// Blank line between art and hints
	lines = append(lines, "")

	// Center each hint
	for _, hint := range hints {
		padLen := (m.width - len(hint)) / 2
		if padLen < 0 {
			padLen = 0
		}
		lines = append(lines, strings.Repeat(" ", padLen)+hintStyle.Render(hint))
	}

	// Vertically center the block
	totalLines := len(lines)
	availableH := m.height - StatusBarHeight
	topPad := (availableH - totalLines) / 2
	if topPad < 0 {
		topPad = 0
	}

	var sb strings.Builder
	for i := 0; i < topPad; i++ {
		sb.WriteByte('\n')
	}
	sb.WriteString(strings.Join(lines, "\n"))

	totalContentLines := topPad + totalLines
	remainingLines := availableH - totalContentLines
	if remainingLines > 0 {
		sb.WriteString(strings.Repeat("\n", remainingLines))
	}

	return sb.String()
}

func (m model) renderHelpPopup() string {
	type entry struct{ key, desc string }
	commands := []entry{
		{"?", "TOGGLE HELP MENU"},
		{"i + i", "ENTER INSERT MODE"},
		{"", ""},
		{"esc + esc", "ENTER PANE MODE"},
		{"n", "CREATE TERMINAL PANE"},
		{"h/j/k/l / arrows", "FOCUS LEFT/DOWN/UP/RIGHT"},
		{"q", "CLOSE FOCUSED PANE"},
		{"1-5", "SWITCH WORKSPACE"},
		{"", ""},
		{"ctrl+q", "QUIT TERMINALSCALE"},
	}

	maxKeyLen := 0
	for _, c := range commands {
		if len(c.key) > maxKeyLen {
			maxKeyLen = len(c.key)
		}
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(colorLightYellow).
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(colorLightYellow)

	descStyle := lipgloss.NewStyle().
		Foreground(colorWhite)

	var lines []string
	lines = append(lines, titleStyle.Render("KEYBINDINGS"))
	lines = append(lines, "")
	for _, c := range commands {
		key := fmt.Sprintf("%-*s", maxKeyLen, c.key)
		lines = append(lines, keyStyle.Render(key)+"  "+descStyle.Render(c.desc))
	}

	popupStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(colorDarkGray).
		Padding(1, 3)

	popup := popupStyle.Render(strings.Join(lines, "\n"))

	// Center the popup over the screen
	popupW := lipgloss.Width(popup)
	popupH := lipgloss.Height(popup)
	x := (m.width - popupW) / 2
	y := (m.height - StatusBarHeight - popupH) / 2

	leftPad := strings.Repeat(" ", x)
	topPad := strings.Repeat("\n", y)

	centered := topPad + leftPad + strings.ReplaceAll(popup, "\n", "\n"+leftPad)

	availableH := m.height - StatusBarHeight
	bottomPad := availableH - y - popupH
	if bottomPad > 0 {
		centered += strings.Repeat("\n", bottomPad)
	}

	return centered
}
