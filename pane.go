package main

import "github.com/hinshun/vt10x"

type Pane struct {
	pty    *PTY
	term   vt10x.Terminal
	x      int
	y      int
	width  int
	height int
}
