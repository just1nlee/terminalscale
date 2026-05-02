package main

import (
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

func main() {
	cmd := exec.Command("bash")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		panic(err)
	}
	defer ptmx.Close()

	go io.Copy(os.Stdout, ptmx)

	io.Copy(ptmx, os.Stdin)
}
