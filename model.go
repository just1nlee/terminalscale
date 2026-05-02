package main

import (
	"fmt"
	"image/color"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

func (m model) Init() tea.Cmd {
	return readPTY(m.pty.Master)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		key := msg.Key()
		if key.Text != "" {
			// Printable character
			m.pty.Master.Write([]byte(key.Text))
		} else {
			// Special key, translate to raw bytes
			switch key.Code {
			case tea.KeyEnter:
				m.pty.Master.Write([]byte("\r"))
			case tea.KeyBackspace:
				m.pty.Master.Write([]byte("\x7f"))
			case tea.KeyTab:
				m.pty.Master.Write([]byte("\t"))
			case tea.KeyUp:
				m.pty.Master.Write([]byte("\x1b[A"))
			case tea.KeyDown:
				m.pty.Master.Write([]byte("\x1b[B"))
			case tea.KeyRight:
				m.pty.Master.Write([]byte("\x1b[C"))
			case tea.KeyLeft:
				m.pty.Master.Write([]byte("\x1b[D"))
			}
		}
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
			cell := m.term.Cell(x, y)

			ch := cell.Char
			if ch == 0 {
				ch = ' '
			}

			// Add color using lipgloss
			style := lipgloss.NewStyle()
			// Check default foreground, if cell doesn't have explicit color set then use terminal default
			if cell.FG != vt10x.DefaultFG {
				style = style.Foreground(vtColor(cell.FG))
			}
			// Check default background, if cell doesn't have explicit color set then use terminal default
			if cell.BG != vt10x.DefaultBG {
				style = style.Background(vtColor(cell.BG))
			}

			sb.WriteString(style.Render(string(ch)))

		}
		if y < m.height-1 {
			sb.WriteByte('\n')
		}
	}

	// Update view with the built string
	v := tea.NewView(sb.String())
	v.AltScreen = true

	// Pass cursor position from vt10x to bubbletea
	cursor := m.term.Cursor()
	if m.term.CursorVisible() {
		v.Cursor = tea.NewCursor(cursor.X, cursor.Y)
	}

	return v
}

// HELPER FUNCTIONS

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

func vtColor(c vt10x.Color) color.Color {
	if c.ANSI() {
		// 0-15: standard ANSI colors
		return lipgloss.ANSIColor(int(c))
	}
	// 16-255: xterm 256 colors
	return lipgloss.Color(fmt.Sprintf("%d", int(c)))

}
