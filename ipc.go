package main

import (
	"encoding/json"
	"net"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
)

const SockPath = "/tmp/terminalscale.sock"

type IPCRequest struct {
	Action string `json:"action"`
	PaneID int    `json:"pane_id,omitempty"`
}

type IPCPaneInfo struct {
	ID     int `json:"id"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type IPCResponse struct {
	PaneID  int           `json:"pane_id,omitempty"`
	Content string        `json:"content,omitempty"`
	Panes   []IPCPaneInfo `json:"panes,omitempty"`
	Error   string        `json:"error,omitempty"`
}

// ipcMsg injects an IPC request into the Bubble Tea update loop.
// The goroutine waits on respCh for the model to reply.
type ipcMsg struct {
	req    IPCRequest
	respCh chan IPCResponse
}

func serveIPC(prog *tea.Program) {
	os.Remove(SockPath)
	ln, err := net.Listen("unix", SockPath)
	if err != nil {
		return
	}
	defer ln.Close()
	defer os.Remove(SockPath)

	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go handleIPCConn(conn, prog)
	}
}

func handleIPCConn(conn net.Conn, prog *tea.Program) {
	defer conn.Close()

	var req IPCRequest
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		json.NewEncoder(conn).Encode(IPCResponse{Error: err.Error()})
		return
	}

	respCh := make(chan IPCResponse, 1)
	prog.Send(ipcMsg{req: req, respCh: respCh})
	resp := <-respCh
	json.NewEncoder(conn).Encode(resp)
}

// readPanePlain renders a pane's screen as plain text (no ANSI codes).
func readPanePlain(p *Pane) string {
	p.term.Lock()
	defer p.term.Unlock()

	width, height := p.term.Size()
	var sb strings.Builder
	for y := 0; y < height; y++ {
		line := make([]rune, width)
		for x := 0; x < width; x++ {
			ch := p.term.Cell(x, y).Char
			if ch == 0 {
				ch = ' '
			}
			line[x] = ch
		}
		sb.WriteString(strings.TrimRight(string(line), " "))
		if y < height-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}
