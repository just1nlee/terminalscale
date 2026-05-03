package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/term"
)

func main() {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		w, h = 80, 24
	}

	firstPane, err := NewPane(0, 0, w, h)
	if err != nil {
		panic(err)
	}

	m := model{
		panes:   []*Pane{firstPane},
		focused: 0,
		width:   w,
		height:  h,
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println("error running program:", err)
		os.Exit(1)
	}
}
