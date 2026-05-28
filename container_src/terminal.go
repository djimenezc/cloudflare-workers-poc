package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/creack/pty"
	"nhooyr.io/websocket"
)

type ptyMsg struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
}

func handleTerminal(workspace string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Printf("ws accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusInternalError, "closing")

		shell := envOr("SHELL", "/bin/bash")
		cmd := exec.Command(shell, "-l")
		cmd.Dir = workspace
		cmd.Env = append(os.Environ(), "TERM=xterm-256color", "PS1=\\w $ ")

		ptmx, err := pty.Start(cmd)
		if err != nil {
			log.Printf("pty start: %v", err)
			return
		}
		defer func() {
			_ = ptmx.Close()
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		// keepalive ping every 20s — survives Cloudflare's idle-close on quiet sockets
		go func() {
			t := time.NewTicker(20 * time.Second)
			defer t.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					pctx, pcancel := context.WithTimeout(ctx, 10*time.Second)
					if err := conn.Ping(pctx); err != nil {
						pcancel()
						cancel()
						return
					}
					pcancel()
				}
			}
		}()

		// pty -> ws
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := ptmx.Read(buf)
				if n > 0 {
					if werr := conn.Write(ctx, websocket.MessageBinary, buf[:n]); werr != nil {
						cancel()
						return
					}
				}
				if err != nil {
					if err != io.EOF {
						log.Printf("pty read: %v", err)
					}
					cancel()
					return
				}
			}
		}()

		// ws -> pty
		for {
			typ, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			if typ == websocket.MessageText {
				var msg ptyMsg
				if err := json.Unmarshal(data, &msg); err == nil && msg.Type == "resize" {
					_ = pty.Setsize(ptmx, &pty.Winsize{Cols: msg.Cols, Rows: msg.Rows})
					continue
				}
			}
			if _, err := ptmx.Write(data); err != nil {
				return
			}
		}
	}
}
