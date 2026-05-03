package main

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

type PTY struct {
	Master   *os.File
	oldState *term.State
}

func NewPTY(cmd *exec.Cmd) (*PTY, error) {
	master, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	return &PTY{Master: master}, nil
}

func (p *PTY) SaveState() error {
	// Switch to raw mode when running programs
	oldState, err := term.MakeRaw(int(os.Stdin.Fd())) // Saves cooked mode state
	if err != nil {
		return err
	}
	p.oldState = oldState
	return nil
}

func (p *PTY) RestoreState() {
	if p.oldState != nil {
		// Switch back to cooked mode after program ends
		term.Restore(int(os.Stdin.Fd()), p.oldState) // Restores cooked mode state
	}
}

func (p *PTY) HandleResize() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			size, err := pty.GetsizeFull(os.Stdin)
			if err != nil {
				return
			}
			pty.Setsize(p.Master, size)
		}
	}()
	ch <- syscall.SIGWINCH
}

func (p *PTY) Close() {
	p.Master.Close()
}
