package main

import (
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

const (
	MinWidth      = 80
	MinHeight     = 24
	MinPaneWidth  = 40
	MinPaneHeight = 12
)

type Pane struct {
	pty    *PTY
	term   vt10x.Terminal
	x      int
	y      int
	width  int
	height int
}

func NewPane(x, y, width, height int) (*Pane, error) {
	// SHELL to run
	cmd := exec.Command("bash")
	// Set TERM environment variable
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	p, err := NewPTY(cmd)
	if err != nil {
		return nil, err
	}

	term := vt10x.New(
		vt10x.WithSize(width, height),
		vt10x.WithWriter(io.Discard),
	)

	pty.Setsize(p.Master, &pty.Winsize{
		Rows: uint16(height),
		Cols: uint16(width),
	})

	return &Pane{
		pty:    p,
		term:   term,
		x:      x,
		y:      y,
		width:  width,
		height: height,
	}, nil
}

func (p *Pane) Resize(x, y, width, height int) {
	p.x = x
	p.y = y
	p.width = width
	p.height = height
	p.term.Resize(width, height)            // Tells vt10x to resize
	pty.Setsize(p.pty.Master, &pty.Winsize{ // Tells the SHELL through the PTY to resize
		Rows: uint16(height),
		Cols: uint16(width),
	})
}

func (p *Pane) Close() {
	p.pty.Close()
}
