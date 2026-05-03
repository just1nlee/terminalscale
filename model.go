package main

import (
	"fmt"
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/hinshun/vt10x"
)

type workspace struct {
	panes   []*Pane
	focused int
}

type model struct {
	workspaces        [10]workspace
	currentWorkspace  int
	width             int
	height            int
	paneMode          bool
	lastPaneModeKey   string
	lastInsertModeKey string
	showHelp          bool
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

func (m *model) ws() *workspace {
	return &m.workspaces[m.currentWorkspace]
}

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, p := range m.workspaces[m.currentWorkspace].panes {
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
		if !m.paneMode && key == "esc" && m.lastInsertModeKey == "esc" {
			m.paneMode = true
			m.lastInsertModeKey = ""
			return m, nil
		}
		if m.paneMode {
			if m.showHelp {
				if key == "?" || key == "esc" {
					m.showHelp = false
				}
				m.lastPaneModeKey = key
				return m, nil
			}

			// Toggle insert mode
			if key == "i" && m.lastPaneModeKey == "i" && len(m.ws().panes) > 0 {
				m.paneMode = false
				m.showHelp = false
				m.lastPaneModeKey = ""
				return m, nil
			}
			m.lastPaneModeKey = key
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
			case "?":
				m.showHelp = !m.showHelp
				return m, nil
			case "1", "2", "3", "4", "5", "6", "7", "8", "9":
				m.currentWorkspace = int(key[0] - '1')
				m.recalculateLayout()
			case "0":
				m.currentWorkspace = 9
				m.recalculateLayout()
			}
			return m, nil
		}

		// Track last key for esc+esc detection in insert mode
		m.lastInsertModeKey = key

		if len(m.ws().panes) == 0 {
			return m, nil
		}

		// Send input to focused pane
		focused := m.ws().panes[m.ws().focused]
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
		// Search all workspaces, background panes still produce output
		for wi := range m.workspaces {
			for idx, p := range m.workspaces[wi].panes {
				if p == msg.pane {
					m.workspaces[wi].panes[idx].term.Write([]byte(msg.data))
					return m, readPane(m.workspaces[wi].panes[idx])
				}
			}
		}
		return m, nil // pane was closed, discard in-flight output
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalculateLayout()
	case ipcMsg:
		resp, cmd := m.handleIPC(msg.req)
		msg.respCh <- resp
		return m, cmd
	}
	return m, nil
}

func (m *model) handleIPC(req IPCRequest) (IPCResponse, tea.Cmd) {
	switch req.Action {
	case "create_pane":
		prevLen := len(m.ws().panes)
		cmd := m.splitPane()
		ws := m.ws()
		if len(ws.panes) > prevLen {
			return IPCResponse{PaneID: ws.panes[len(ws.panes)-1].ID}, cmd
		}
		return IPCResponse{Error: "cannot create pane (max 4 or terminal too small)"}, nil

	case "read_pane":
		for wi := range m.workspaces {
			for _, p := range m.workspaces[wi].panes {
				if p.ID == req.PaneID {
					return IPCResponse{PaneID: p.ID, Content: readPanePlain(p)}, nil
				}
			}
		}
		return IPCResponse{Error: "pane not found"}, nil

	case "close_pane":
		for wi := range m.workspaces {
			for i, p := range m.workspaces[wi].panes {
				if p.ID == req.PaneID {
					saved := m.currentWorkspace
					m.currentWorkspace = wi
					m.ws().focused = i
					m.closePane()
					m.currentWorkspace = saved
					return IPCResponse{PaneID: req.PaneID}, nil
				}
			}
		}
		return IPCResponse{Error: "pane not found"}, nil

	case "list_panes":
		var infos []IPCPaneInfo
		for wi := range m.workspaces {
			for _, p := range m.workspaces[wi].panes {
				infos = append(infos, IPCPaneInfo{ID: p.ID, Width: p.width, Height: p.height})
			}
		}
		return IPCResponse{Panes: infos}, nil

	default:
		return IPCResponse{Error: "unknown action: " + req.Action}, nil
	}
}

func (m *model) splitPane() tea.Cmd {
	extraWidth := paneExtraW()
	extraHeight := paneExtraH()
	ws := m.ws()

	switch len(ws.panes) {
	case 0:
		p, err := NewPane(0, 0, m.width-extraWidth, m.height-extraHeight-StatusBarHeight)
		if err != nil {
			return nil
		}
		ws.panes = append(ws.panes, p)
		ws.focused = len(ws.panes) - 1
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
		ws.panes = append(ws.panes, p)
		ws.focused = len(ws.panes) - 1
		m.recalculateLayout()
		return readPane(p)

	case 2:
		if m.height/2 < MinPaneHeight {
			return nil
		}
		halfH := (m.height - extraHeight*2) / 2
		leftX := ws.panes[0].x
		leftW := ws.panes[0].width
		p, err := NewPane(leftX, halfH+extraHeight, leftW, m.height-extraHeight*2-halfH)
		if err != nil {
			return nil
		}
		ws.panes = append(ws.panes, p)
		ws.focused = len(ws.panes) - 1
		m.recalculateLayout()
		return readPane(p)

	case 3:
		if m.height/2 < MinPaneHeight {
			return nil
		}
		halfH := (m.height - extraHeight*2) / 2
		rightX := ws.panes[1].x
		rightW := ws.panes[1].width
		p, err := NewPane(rightX, halfH+extraHeight, rightW, m.height-extraHeight*2-halfH)
		if err != nil {
			return nil
		}
		ws.panes = append(ws.panes, p)
		ws.focused = len(ws.panes) - 1
		m.recalculateLayout()
		return readPane(p)

	default:
		return nil
	}
}

func (m *model) closePane() {
	ws := m.ws()
	if len(ws.panes) == 0 {
		return
	}
	ws.panes[ws.focused].Close()

	copy(ws.panes[ws.focused:], ws.panes[ws.focused+1:]) // Shifts elements left
	ws.panes[len(ws.panes)-1] = nil                      // nil the last slot
	ws.panes = ws.panes[:len(ws.panes)-1]                // Shrink the array
	if len(ws.panes) == 0 {
		ws.focused = 0
	} else if ws.focused >= len(ws.panes) {
		ws.focused = len(ws.panes) - 1
	}

	m.recalculateLayout()
}

func (m *model) focusLeft() {
	ws := m.ws()
	if len(ws.panes) == 0 {
		return
	}
	focused := ws.panes[ws.focused]
	for i, p := range ws.panes {
		if p.x < focused.x && p.y < focused.y+focused.height && p.y+p.height > focused.y {
			ws.focused = i
			return
		}
	}
}

func (m *model) focusRight() {
	ws := m.ws()
	if len(ws.panes) == 0 {
		return
	}
	focused := ws.panes[ws.focused]
	for i, p := range ws.panes {
		if p.x > focused.x && p.y < focused.y+focused.height && p.y+p.height > focused.y {
			ws.focused = i
			return
		}
	}
}

func (m *model) focusUp() {
	ws := m.ws()
	if len(ws.panes) == 0 {
		return
	}
	focused := ws.panes[ws.focused]
	for i, p := range ws.panes {
		if p.y < focused.y && p.x < focused.x+focused.width && p.x+p.width > focused.x {
			ws.focused = i
			return
		}
	}
}

func (m *model) focusDown() {
	ws := m.ws()
	if len(ws.panes) == 0 {
		return
	}
	focused := ws.panes[ws.focused]
	for i, p := range ws.panes {
		if p.y > focused.y && p.x < focused.x+focused.width && p.x+p.width > focused.x {
			ws.focused = i
			return
		}
	}
}

func (m *model) recalculateLayout() {
	extraWidth := paneExtraW()
	extraHeight := paneExtraH()
	h := m.height - StatusBarHeight
	panes := m.ws().panes

	switch len(panes) {
	case 1:
		panes[0].Resize(0, 0, m.width-extraWidth, h-extraHeight)
	case 2:
		half := (m.width - extraWidth*2) / 2
		panes[0].Resize(0, 0, half, h-extraHeight)
		panes[1].Resize(half+extraWidth, 0, m.width-extraWidth*2-half, h-extraHeight)
	case 3:
		halfW := (m.width - extraWidth*2) / 2
		halfH := (h - extraHeight*2) / 2
		panes[0].Resize(0, 0, halfW, halfH)
		panes[1].Resize(halfW+extraWidth, 0, m.width-extraWidth*2-halfW, h-extraHeight)
		panes[2].Resize(0, halfH+extraHeight, halfW, h-extraHeight*2-halfH)
	case 4:
		halfW := (m.width - extraWidth*2) / 2
		halfH := (h - extraHeight*2) / 2
		panes[0].Resize(0, 0, halfW, halfH)
		panes[1].Resize(halfW+extraWidth, 0, m.width-extraWidth*2-halfW, halfH)
		panes[2].Resize(0, halfH+extraHeight, halfW, h-extraHeight*2-halfH)
		panes[3].Resize(halfW+extraWidth, halfH+extraHeight, m.width-extraWidth*2-halfW, h-extraHeight*2-halfH)
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
		Foreground(colorLightYellow).
		Bold(true).
		Render("TERMINALSCALE")

	// Workspace 0-9 indicator
	activeWsStyle := lipgloss.NewStyle().
		Background(colorLightYellow).
		Foreground(colorBackground).
		Bold(true)
	var wsParts []string
	for i := 0; i < 10; i++ {
		label := fmt.Sprintf(" %d ", (i+1)%10) // 1-9, then 0
		if i == m.currentWorkspace {
			wsParts = append(wsParts, activeWsStyle.Render(label))
		} else {
			wsParts = append(wsParts, statusBarStyle.Render(label))
		}
	}
	right := strings.Join(wsParts, "")

	left := statusBarStyle.Render(mode)
	leftW := lipgloss.Width(left)
	centerW := lipgloss.Width(center)
	rightW := lipgloss.Width(right)

	centerStart := (m.width - centerW) / 2
	leftPad := centerStart - leftW
	if leftPad < 0 {
		leftPad = 0
	}
	rightPad := m.width - centerStart - centerW - rightW
	if rightPad < 0 {
		rightPad = 0
	}

	bar := left +
		statusBarStyle.Render(strings.Repeat(" ", leftPad)) +
		center +
		statusBarStyle.Render(strings.Repeat(" ", rightPad)) +
		right

	return bar
}

func (m model) View() tea.View {
	// Check min size
	if m.width < MinWidth || m.height < MinHeight {
		v := tea.NewView(fmt.Sprintf("ERROR: Terminal too small. Minimum size: %dx%d", MinWidth, MinHeight))
		v.AltScreen = true
		return v
	}

	ws := m.workspaces[m.currentWorkspace]

	// Combine pane renders using lipgloss by joining pane strings side by side
	var rendered []string
	for i, p := range ws.panes {
		content := renderPane(p)
		if i == ws.focused {
			rendered = append(rendered, focusedPaneStyle.Render(content))
		} else {
			rendered = append(rendered, unfocusedPaneStyle.Render(content))
		}
	}

	var content string
	switch len(ws.panes) {
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

	var screen string
	if len(ws.panes) == 0 {
		screen = m.renderHomeScreen()
	} else {
		screen = content
	}

	if m.showHelp {
		screen = m.renderHelpPopup()
	}

	// Update view with the built string
	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, screen, m.renderStatusBar()))
	v.AltScreen = true

	// Pass cursor position to focused pane from vt10x to bubbletea
	if len(ws.panes) > 0 {
		focused := ws.panes[ws.focused]
		focused.term.Lock()
		cursor := focused.term.Cursor()
		visible := focused.term.CursorVisible()
		focused.term.Unlock()

		if visible {
			v.Cursor = tea.NewCursor(
				focused.x+CursorOffsetX+cursor.X,
				focused.y+CursorOffsetY+cursor.Y,
			)
		}
	}

	return v
}
