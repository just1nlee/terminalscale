package main

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"
)

type model struct {
	pty    *PTY
	buffer string
	width  int
	height int
}

type ptyOutput struct {
	data string
}

// Cmd that reads up to 4096 bytes from PTY, returns as a Msg for Update() to read
// Takes PTY handler type as parameters
func readPTY(master *os.File) tea.Cmd {
	return func() tea.Msg {
		buf := make([]byte, 4096)
		n, err := master.Read(buf)
		if err != nil {
			return nil
		}
		return ptyOutput{data: string(buf[:n])}
	}
}

func (m model) Init() tea.Cmd {
	return readPTY(m.pty.Master)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		m.pty.Master.Write([]byte(msg.String()))
	case ptyOutput:
		m.buffer += msg.data
		return m, readPTY(m.pty.Master)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		pty.Setsize(m.pty.Master, &pty.Winsize{
			Rows: uint16(msg.Height),
			Cols: uint16(msg.Width),
		})
	}
	return m, nil
}

func (m model) View() string {
	return m.buffer
}
