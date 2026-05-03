package main

import (
	"strings"

	"charm.land/lipgloss/v2"
)

var asciiArt = []string{
	` _____ _____ ____  ___ __  __ ___ _   _    _    _     ____   ____    _    _     _____ `,
	`|_   _| ____|  _ \|_ _|  \/  |_ _| \ | |  / \  | |   / ___| / ___|  / \  | |   | ____|`,
	`  | | |  _| | |_) || || |\/| || ||  \| | / _ \ | |   \___ \| |     / _ \ | |   |  _|  `,
	`  | | | |___|  _ < | || |  | || || |\  |/ ___ \| |___ ___) | |___ / ___ \| |___| |___ `,
	`  |_| |_____|_| \_\___|_|  |_|___|_| \_/_/   \_\_____|____/ \____/_/   \_\_____|_____|`,
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
