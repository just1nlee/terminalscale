package main

import (
	"fmt"
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/hinshun/vt10x"
)

type model struct {
	panes    []*Pane
	focused  int
	width    int
	height   int
	paneMode bool
	lastKey  string
}

type ptyOutput struct {
	pane *Pane
	data string
}

// Cmd that reads up to 4096 bytes from PTY, returns as a Msg for Update() to read
func readPane(p *Pane) tea.Cmd {
	return func() tea.Msg {
		buf := make([]byte, 4096)
		n, err := p.pty.Master.Read(buf)
		if err != nil {
			return nil
		}
		return ptyOutput{pane: p, data: string(buf[:n])}
	}
}

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, p := range m.panes {
		cmds = append(cmds, readPane(p))
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		key := msg.String()

		if key == "ctrl+q" {
			return m, tea.Quit
		}

		// Toggle pane mode
		if !m.paneMode && msg.String() == "esc" && m.lastKey == "esc" {
			m.paneMode = true
			m.lastKey = ""
			return m, nil
		}
		if m.paneMode {
			// Toggle insert mode (only when a pane exists to receive input)
			if key == "i" && m.lastKey == "i" && len(m.panes) > 0 {
				m.paneMode = false
				m.lastKey = ""
				return m, nil
			}
			m.lastKey = key
			switch key {
			case "k", "w":
				m.focusUp()
			case "h", "a":
				m.focusLeft()
			case "j", "s":
				m.focusDown()
			case "l", "d":
				m.focusRight()
			case "n":
				cmd := m.splitPane()
				return m, cmd
			case "q":
				m.closePane()
			}
			return m, nil
		}

		// Record key for double-press detection for insert mode
		m.lastKey = key

		if len(m.panes) == 0 {
			return m, nil
		}

		// Send input to focused pane
		focused := m.panes[m.focused]
		k := msg.Key()
		if k.Text != "" {
			// Printable character
			focused.pty.Master.Write([]byte(k.Text))
		} else {
			// Special key, translate to raw bytes
			switch k.Code {
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
			case tea.KeyEscape:
				focused.pty.Master.Write([]byte("\x1b"))

			}
		}
	case ptyOutput:
		idx := -1
		for i, p := range m.panes {
			if p == msg.pane {
				idx = i
				break
			}
		}
		if idx == -1 {
			return m, nil // pane was closed, discard in-flight output
		}
		m.panes[idx].term.Write([]byte(msg.data)) // feeds raw bytes from PTY into vt10x
		return m, readPane(m.panes[idx])
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Resize vt10x terminal window also
		m.recalculateLayout()
	}
	return m, nil
}

func (m *model) splitPane() tea.Cmd {
	extraWidth := paneExtraW()
	extraHeight := paneExtraH()

	switch len(m.panes) {
	case 0:
		p, err := NewPane(0, 0, m.width-extraWidth, m.height-extraHeight-StatusBarHeight)
		if err != nil {
			return nil
		}
		m.panes = append(m.panes, p)
		m.recalculateLayout()
		return readPane(p)

	case 1:
		if m.width/2 < MinPaneWidth {
			return nil
		}
		half := (m.width - extraWidth*2) / 2
		p, err := NewPane(half+extraWidth, 0, m.width-extraWidth*2-half, m.height-extraHeight)
		if err != nil {
			return nil
		}
		m.panes = append(m.panes, p)
		m.recalculateLayout()
		return readPane(p)

	case 2:
		if m.height/2 < MinPaneHeight {
			return nil
		}
		halfH := (m.height - extraHeight*2) / 2
		leftX := m.panes[0].x
		leftW := m.panes[0].width
		p, err := NewPane(leftX, halfH+extraHeight, leftW, m.height-extraHeight*2-halfH)
		if err != nil {
			return nil
		}
		m.panes = append(m.panes, p)
		m.recalculateLayout()
		return readPane(p)

	case 3:
		if m.height/2 < MinPaneHeight {
			return nil
		}
		halfH := (m.height - extraHeight*2) / 2
		rightX := m.panes[1].x
		rightW := m.panes[1].width
		p, err := NewPane(rightX, halfH+extraHeight, rightW, m.height-extraHeight*2-halfH)
		if err != nil {
			return nil
		}
		m.panes = append(m.panes, p)
		m.recalculateLayout()
		return readPane(p)

	default:
		return nil
	}
}

func (m *model) closePane() {
	if len(m.panes) == 0 {
		return
	}
	m.panes[m.focused].Close()

	copy(m.panes[m.focused:], m.panes[m.focused+1:]) // Shifts elements left
	m.panes[len(m.panes)-1] = nil                    // nil the last slot
	m.panes = m.panes[:len(m.panes)-1]               // Shrink the array
	if len(m.panes) == 0 {
		m.focused = 0
	} else if m.focused >= len(m.panes) {
		m.focused = len(m.panes) - 1
	}

	m.recalculateLayout()
}

func (m *model) focusLeft() {
	if len(m.panes) == 0 {
		return
	}
	focused := m.panes[m.focused]
	for i, p := range m.panes {
		if p.x < focused.x && p.y < focused.y+focused.height && p.y+p.height > focused.y {
			m.focused = i
			return
		}
	}
}

func (m *model) focusRight() {
	if len(m.panes) == 0 {
		return
	}
	focused := m.panes[m.focused]
	for i, p := range m.panes {
		if p.x > focused.x && p.y < focused.y+focused.height && p.y+p.height > focused.y {
			m.focused = i
			return
		}
	}
}

func (m *model) focusUp() {
	if len(m.panes) == 0 {
		return
	}
	focused := m.panes[m.focused]
	for i, p := range m.panes {
		if p.y < focused.y && p.x < focused.x+focused.width && p.x+p.width > focused.x {
			m.focused = i
			return
		}
	}
}

func (m *model) focusDown() {
	if len(m.panes) == 0 {
		return
	}
	focused := m.panes[m.focused]
	for i, p := range m.panes {
		if p.y > focused.y && p.x < focused.x+focused.width && p.x+p.width > focused.x {
			m.focused = i
			return
		}
	}
}

func (m *model) recalculateLayout() {
	extraWidth := paneExtraW()
	extraHeight := paneExtraH()
	h := m.height - StatusBarHeight

	switch len(m.panes) {
	case 1:
		m.panes[0].Resize(0, 0, m.width-extraWidth, h-extraHeight)
	case 2:
		half := (m.width - extraWidth*2) / 2
		m.panes[0].Resize(0, 0, half, h-extraHeight)
		m.panes[1].Resize(half+extraWidth, 0, m.width-extraWidth*2-half, h-extraHeight)
	case 3:
		halfW := (m.width - extraWidth*2) / 2
		halfH := (h - extraHeight*2) / 2
		m.panes[0].Resize(0, 0, halfW, halfH)
		m.panes[1].Resize(halfW+extraWidth, 0, m.width-extraWidth*2-halfW, h-extraHeight)
		m.panes[2].Resize(0, halfH+extraHeight, halfW, h-extraHeight*2-halfH)
	case 4:
		halfW := (m.width - extraWidth*2) / 2
		halfH := (h - extraHeight*2) / 2
		m.panes[0].Resize(0, 0, halfW, halfH)
		m.panes[1].Resize(halfW+extraWidth, 0, m.width-extraWidth*2-halfW, halfH)
		m.panes[2].Resize(0, halfH+extraHeight, halfW, h-extraHeight*2-halfH)
		m.panes[3].Resize(halfW+extraWidth, halfH+extraHeight, m.width-extraWidth*2-halfW, h-extraHeight*2-halfH)
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

	width, height := p.term.Size()

	// String builder reads from vt10x's internal 2D cell grid
	var sb strings.Builder
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
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

		if y < height-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func (m model) renderStatusBar() string {
	var mode string
	if m.paneMode {
		mode = " PANE MODE "
	} else {
		mode = " -- INSERT -- "
	}

	center := lipgloss.NewStyle().
		Background(lipgloss.Color("#1a1a1a")).
		Foreground(colorGreenDim).
		Bold(true).
		Render("TERMINALSCALE")

	left := statusBarStyle.Render(mode)

	leftW := lipgloss.Width(left)
	centerW := lipgloss.Width(center)

	leftPad := (m.width / 2) - leftW - (centerW / 2)
	rightPad := m.width - leftW - leftPad - centerW

	bar := left +
		statusBarStyle.Render(strings.Repeat(" ", leftPad)) +
		center +
		statusBarStyle.Render(strings.Repeat(" ", rightPad))

	return bar
}

func (m model) View() tea.View {
	// Check min size
	if m.width < MinWidth || m.height < MinHeight {
		v := tea.NewView(fmt.Sprintf("Terminal too small. Minimum size: %dx%d", MinWidth, MinHeight))
		v.AltScreen = true
		return v
	}

	if len(m.panes) == 0 {
		v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, m.renderHomeScreen(), m.renderStatusBar()))
		v.AltScreen = true
		return v
	}

	// Combine pane renders using lipgloss by joing joining pane strings side by side
	var rendered []string
	for i, p := range m.panes {
		content := renderPane(p)
		if i == m.focused {
			rendered = append(rendered, focusedPaneStyle.Render(content))
		} else {
			rendered = append(rendered, unfocusedPaneStyle.Render(content))
		}
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
	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, content, m.renderStatusBar()))

	v.AltScreen = true

	// Pass cursor position to focused pane from vt10x to bubbletea
	focused := m.panes[m.focused]
	focused.term.Lock()
	cursor := focused.term.Cursor()
	visible := focused.term.CursorVisible()
	focused.term.Unlock()

	// Calculate cursor position
	if visible {
		v.Cursor = tea.NewCursor(
			focused.x+CursorOffsetX+cursor.X,
			focused.y+CursorOffsetY+cursor.Y,
		)
	}

	return v
}
