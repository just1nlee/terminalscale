package main

import (
	"io"
	"os"
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

	// PTY Output
	go io.Copy(os.Stdout, ptmx.Master)
	// PTY Input
	io.Copy(ptmx.Master, os.Stdin)
}
