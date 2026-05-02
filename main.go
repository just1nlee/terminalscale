package main

import (
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/hinshun/vt10x"
)

func main() {
	// PTY pair handler
	ptmx, err := NewPTY()
	if err != nil {
		panic(err)
	}

	defer ptmx.Close()

	ptmx.HandleResize()

	term := vt10x.New(vt10x.WithSize(80, 24), vt10x.WithWriter(io.Discard))

	m := model{
		pty:  ptmx,
		term: term,
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println("error running program:", err)
		os.Exit(1)
	}
}
