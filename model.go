package main

import (
	"fmt"
	"image/color"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/hinshun/vt10x"
)

type model struct {
	panes   []*Pane
	focused int
	width   int
	height  int
}

type ptyOutput struct {
	index int
	data  string
}

// Cmd that reads up to 4096 bytes from PTY, returns as a Msg for Update() to read
// Takes PTY handler type as parameters
func readPane(index int, master *os.File) tea.Cmd {
	return func() tea.Msg {
		buf := make([]byte, 4096)
		n, err := master.Read(buf)
		if err != nil {
			return nil
		}
		return ptyOutput{index: index, data: string(buf[:n])}
	}
}

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for i, p := range m.panes {
		cmds = append(cmds, readPane(i, p.pty.Master))
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "ctrl+n" {
			m.splitPane()
			return m, nil
		}
		// Cycle focus
		if msg.String() == "ctrl+]" {
			m.focused = (m.focused + 1) % len(m.panes)
			return m, nil
		}
		// Send input to focused pane
		focused := m.panes[m.focused]
		key := msg.Key()
		if key.Text != "" {
			// Printable character
			focused.pty.Master.Write([]byte(key.Text))
		} else {
			// Special key, translate to raw bytes
			switch key.Code {
			case tea.KeyEnter:
				focused.pty.Master.Write([]byte("\r"))
			case tea.KeyBackspace:
				focused.pty.Master.Write([]byte("\x7f"))
			case tea.KeyTab:
				focused.pty.Master.Write([]byte("\t"))
			case tea.KeyUp:
				focused.pty.Master.Write([]byte("\x1b[A"))
			case tea.KeyDown:
				focused.pty.Master.Write([]byte("\x1b[B"))
			case tea.KeyRight:
				focused.pty.Master.Write([]byte("\x1b[C"))
			case tea.KeyLeft:
				focused.pty.Master.Write([]byte("\x1b[D"))
			}
		}
	case ptyOutput:
		m.panes[msg.index].term.Write([]byte(msg.data)) // feeds raw bytes from PTY into vt10x
		return m, readPane(msg.index, m.panes[msg.index].pty.Master)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Resize vt10x terminal window also
		m.recalculateLayout()
	}
	return m, nil
}

func (m *model) splitPane() {
	switch len(m.panes) {
	case 1:
		// Check min size
		if m.width/2 < MinPaneWidth {
			return
		}
		half := m.width / 2
		m.panes[0].Resize(0, 0, half, m.height)
		p, err := NewPane(half, 0, m.width-half, m.height)
		if err != nil {
			return
		}
		m.panes = append(m.panes, p)

	case 2:
		// Split right pane vertically
		if m.height/2 < MinPaneHeight {
			return
		}
		half := m.height / 2
		rightX := m.panes[1].x
		rightW := m.panes[1].width
		m.panes[1].Resize(rightX, 0, rightW, half)
		p, err := NewPane(rightX, half, rightW, m.height-half)
		if err != nil {
			return
		}
		m.panes = append(m.panes, p)
	case 3:
		// Split left pane vertically
		if m.height/2 < MinPaneHeight {
			return
		}
		half := m.height / 2
		m.panes[0].Resize(0, 0, m.panes[0].width, half)
		p, err := NewPane(0, half, m.panes[0].width, m.height-half)
		if err != nil {
			return
		}
		m.panes = append(m.panes, p)

	case 4:
	}
}

func (m *model) recalculateLayout() {
	switch len(m.panes) {
	case 1:
		m.panes[0].Resize(0, 0, m.width, m.height)
	case 2:
		half := m.width / 2
		m.panes[0].Resize(0, 0, half, m.height)
		m.panes[1].Resize(half, 0, m.width-half, m.height)
	case 3:
		half := m.width / 2
		halfH := m.height / 2
		m.panes[0].Resize(0, 0, half, halfH)
		m.panes[1].Resize(half, 0, m.width-half, m.height)
		m.panes[2].Resize(0, halfH, half, m.height-halfH)
	case 4:
		halfW := m.width / 2
		halfH := m.height / 2
		m.panes[0].Resize(0, 0, halfW, halfH)
		m.panes[1].Resize(halfW, 0, m.width-halfW, halfH)
		m.panes[2].Resize(0, halfH, halfW, m.height-halfH)
		m.panes[3].Resize(halfW, halfH, m.width-halfW, m.height-halfH)
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

func renderPane(p *Pane) string {
	p.term.Lock()
	defer p.term.Unlock()

	// String builder reads from vt10x's internal 2D cell grid
	var sb strings.Builder
	for y := 0; y < p.height; y++ {
		for x := 0; x < p.width; x++ {
			cell := p.term.Cell(x, y)
			ch := cell.Char
			if ch == 0 {
				ch = ' '
			}

			// Add color and mode using lipgloss
			style := lipgloss.NewStyle()
			// Check default foreground, if cell doesn't have explicit color set then use terminal default
			if cell.FG != vt10x.DefaultFG {
				style = style.Foreground(vtColor(cell.FG))
			}
			// Check default background, if cell doesn't have explicit color set then use terminal default
			if cell.BG != vt10x.DefaultBG {
				style = style.Background(vtColor(cell.BG))
			}

			// Check mode
			if cell.Mode&1 != 0 { // AttrReverse
				style = style.Reverse(true)
			}
			if cell.Mode&2 != 0 { // AttrUnderline
				style = style.Underline(true)
			}
			if cell.Mode&4 != 0 { // AttrBold
				style = style.Bold(true)
			}
			if cell.Mode&16 != 0 { // AttrItalic
				style = style.Italic(true)
			}
			sb.WriteString(style.Render(string(ch)))
		}

		if y < p.height-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func (m model) View() tea.View {
	// Check min size
	if m.width < MinWidth || m.height < MinHeight {
		v := tea.NewView(fmt.Sprintf("Terminal too small. Minimum size: %dx%d", MinWidth, MinHeight))
		v.AltScreen = true
		return v
	}

	// Renders pane into a grid of lines
	screen := make([][]rune, m.height)
	for i := range screen {
		screen[i] = make([]rune, m.width)
		for j := range screen[i] {
			screen[i][j] = ' '
		}
	}

	// Combine pane renders using lipgloss by joing joining pane strings side by side
	var rendered []string
	for _, p := range m.panes {
		rendered = append(rendered, renderPane(p))
	}

	var content string
	switch len(m.panes) {
	case 1:
		content = rendered[0]
	case 2:
		content = lipgloss.JoinHorizontal(lipgloss.Top, rendered[0], rendered[1])
	case 3:
		left := lipgloss.JoinVertical(lipgloss.Left, rendered[0], rendered[2])
		content = lipgloss.JoinHorizontal(lipgloss.Top, left, rendered[1])
	case 4:
		left := lipgloss.JoinVertical(lipgloss.Left, rendered[0], rendered[2])
		right := lipgloss.JoinVertical(lipgloss.Left, rendered[1], rendered[3])
		content = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}

	// Update view with the built string
	v := tea.NewView(content)
	v.AltScreen = true

	// Pass cursor position to focused pane from vt10x to bubbletea
	focused := m.panes[m.focused]
	focused.term.Lock()
	cursor := focused.term.Cursor()
	visible := focused.term.CursorVisible()
	focused.term.Unlock()

	if visible {
		v.Cursor = tea.NewCursor(focused.x+cursor.X, focused.y+cursor.Y)
	}

	return v
}
