package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
)

const sockPath = "/tmp/terminalscale.sock"

// JSON-RPC 2.0 types
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string  `json:"jsonrpc"`
	ID      any     `json:"id,omitempty"`
	Result  any     `json:"result,omitempty"`
	Error   *rpcErr `json:"error,omitempty"`
}

type rpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// IPC types — mirrors ipc.go in the multiplexer
type ipcRequest struct {
	Action string `json:"action"`
	PaneID int    `json:"pane_id,omitempty"`
}

type ipcPaneInfo struct {
	ID     int `json:"id"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type ipcResponse struct {
	PaneID  int           `json:"pane_id,omitempty"`
	Content string        `json:"content,omitempty"`
	Panes   []ipcPaneInfo `json:"panes,omitempty"`
	Error   string        `json:"error,omitempty"`
}

func callIPC(req ipcRequest) (ipcResponse, error) {
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return ipcResponse{}, fmt.Errorf("terminalscale not running: %w", err)
	}
	defer conn.Close()
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return ipcResponse{}, err
	}
	var resp ipcResponse
	return resp, json.NewDecoder(conn).Decode(&resp)
}

var tools = map[string]any{
	"tools": []any{
		map[string]any{
			"name":        "create_pane",
			"description": "Create a new terminal pane in the multiplexer. Returns the new pane ID.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		map[string]any{
			"name":        "read_pane",
			"description": "Read the current screen content of a terminal pane as plain text.",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"pane_id"},
				"properties": map[string]any{
					"pane_id": map[string]any{"type": "integer", "description": "Pane ID returned by create_pane or list_panes"},
				},
			},
		},
		map[string]any{
			"name":        "close_pane",
			"description": "Close a terminal pane.",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"pane_id"},
				"properties": map[string]any{
					"pane_id": map[string]any{"type": "integer", "description": "Pane ID to close"},
				},
			},
		},
		map[string]any{
			"name":        "list_panes",
			"description": "List all open terminal panes across all workspaces.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
	},
}

func handleTool(name string, rawArgs json.RawMessage) (string, error) {
	var args struct {
		PaneID int `json:"pane_id"`
	}
	json.Unmarshal(rawArgs, &args)

	switch name {
	case "create_pane":
		resp, err := callIPC(ipcRequest{Action: "create_pane"})
		if err != nil {
			return "", err
		}
		if resp.Error != "" {
			return resp.Error, nil
		}
		return fmt.Sprintf("Created pane %d", resp.PaneID), nil

	case "read_pane":
		resp, err := callIPC(ipcRequest{Action: "read_pane", PaneID: args.PaneID})
		if err != nil {
			return "", err
		}
		if resp.Error != "" {
			return resp.Error, nil
		}
		return resp.Content, nil

	case "close_pane":
		resp, err := callIPC(ipcRequest{Action: "close_pane", PaneID: args.PaneID})
		if err != nil {
			return "", err
		}
		if resp.Error != "" {
			return resp.Error, nil
		}
		return fmt.Sprintf("Closed pane %d", resp.PaneID), nil

	case "list_panes":
		resp, err := callIPC(ipcRequest{Action: "list_panes"})
		if err != nil {
			return "", err
		}
		if resp.Error != "" {
			return resp.Error, nil
		}
		b, _ := json.MarshalIndent(resp.Panes, "", "  ")
		if resp.Panes == nil {
			return "[]", nil
		}
		return string(b), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func reply(enc *json.Encoder, id any, result any) {
	enc.Encode(rpcResponse{JSONRPC: "2.0", ID: id, Result: result})
}

func replyErr(enc *json.Encoder, id any, code int, msg string) {
	enc.Encode(rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcErr{Code: code, Message: msg}})
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	enc := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		var req rpcRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}
		// Notifications have no id; don't respond
		if req.Method == "notifications/initialized" {
			continue
		}

		switch req.Method {
		case "initialize":
			reply(enc, req.ID, map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "terminalscale-mcp", "version": "1.0.0"},
			})

		case "tools/list":
			reply(enc, req.ID, tools)

		case "tools/call":
			var p struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				replyErr(enc, req.ID, -32600, err.Error())
				continue
			}
			text, err := handleTool(p.Name, p.Arguments)
			if err != nil {
				replyErr(enc, req.ID, -32603, err.Error())
				continue
			}
			reply(enc, req.ID, map[string]any{
				"content": []any{map[string]any{"type": "text", "text": text}},
			})

		default:
			replyErr(enc, req.ID, -32601, "method not found")
		}
	}
}
