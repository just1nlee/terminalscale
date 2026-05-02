package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// PTY pair handler
	ptmx, err := NewPTY()
	if err != nil {
		panic(err)
	}

	err = ptmx.SaveState()
	if err != nil {
		panic(err)
	}
	defer ptmx.Close()

	ptmx.HandleResize()

	m := model{
		pty: ptmx,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("error running program:", err)
		os.Exit(1)
	}
}
