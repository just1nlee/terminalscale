package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/term"
)

func main() {
	// Calculate outer terminal size from OS
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		w, h = 80, 24
	}

	// Model needs to store actual size for layout calculations
	m := model{
		width:    w,
		height:   h,
		paneMode: true,
	}

	p := tea.NewProgram(m)
	go serveIPC(p)
	if _, err := p.Run(); err != nil {
		fmt.Println("error running program:", err)
		os.Exit(1)
	}
}
