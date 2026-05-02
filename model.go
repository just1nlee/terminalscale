package main

import (
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

type model struct {
	pty    *PTY
	term   vt10x.Terminal
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
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		m.pty.Master.Write([]byte(msg.String()))
	case ptyOutput:
		m.term.Write([]byte(msg.data)) // feeds raw bytes from PTY into vt10x
		return m, readPTY(m.pty.Master)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Resize vt10x terminal window also
		m.term.Resize(msg.Width, msg.Height)
		pty.Setsize(m.pty.Master, &pty.Winsize{
			Rows: uint16(msg.Height),
			Cols: uint16(msg.Width),
		})
	}
	return m, nil
}

func (m model) View() tea.View {
	m.term.Lock()
	defer m.term.Unlock()

	// String builder reads from vt10x's internal 2D cell grid
	var sb strings.Builder
	for y := 0; y < m.height; y++ {
		for x := 0; x < m.width; x++ {
			ch := m.term.Cell(x, y).Char
			if ch == 0 {
				sb.WriteRune(' ')
			} else {
				sb.WriteRune(ch)
			}
		}
		if y < m.height-1 {
			sb.WriteByte('\n')
		}
	}

	// Update view with the built string
	v := tea.NewView(sb.String())
	v.AltScreen = true
	return v
}
