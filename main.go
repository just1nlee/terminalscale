package main

import (
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func main() {
	cmd := exec.Command("bash")

	// PTY pair handler
	ptmx, err := pty.Start(cmd)
	if err != nil {
		panic(err)
	}
	defer ptmx.Close()

	// Switch to raw mode when running programs
	oldState, err := term.MakeRaw(int(os.Stdin.Fd())) // Saves cooked mode state
	if err != nil {
		panic(err)
	}
	// Switch back to cooked mode after program ends
	defer term.Restore(int(os.Stdin.Fd()), oldState) // Restores cooked mode state

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			size, err := pty.GetsizeFull(os.Stdin)
			if err != nil {
				return
			}
			pty.Setsize(ptmx, size)
		}
	}()
	ch <- syscall.SIGWINCH

	// PTY Output
	go io.Copy(os.Stdout, ptmx)
	// PTY Input
	io.Copy(ptmx, os.Stdin)
}
