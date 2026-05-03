package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

func main() {
	firstPane, err := NewPane(0, 0, 80, 24)
	if err != nil {
		panic(err)
	}

	m := model{
		panes:   []*Pane{firstPane},
		focused: 0,
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println("error running program:", err)
		os.Exit(1)
	}
}
