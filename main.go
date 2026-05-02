package main

import (
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

func main() {
	cmd := exec.Command("bash")

	// PTY handler
	ptmx, err := pty.Start(cmd)
	if err != nil {
		panic(err)
	}
	defer ptmx.Close()

	// Output
	go io.Copy(os.Stdout, ptmx)
	// Input
	io.Copy(ptmx, os.Stdin)
}
