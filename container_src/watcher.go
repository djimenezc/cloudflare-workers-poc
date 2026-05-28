package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"nhooyr.io/websocket"
)

type Watcher struct {
	root    string
	fs      *fsnotify.Watcher
	mu      sync.Mutex
	clients map[chan fsEvent]struct{}
}

type fsEvent struct {
	Op   string `json:"op"`
	Path string `json:"path"`
}

func newWatcher(root string) (*Watcher, error) {
	fs, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{root: root, fs: fs, clients: map[chan fsEvent]struct{}{}}

	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		if strings.HasPrefix(filepath.Base(path), ".") && path != root {
			return filepath.SkipDir
		}
		return fs.Add(path)
	}); err != nil {
		return nil, err
	}

	go w.loop()
	return w, nil
}

func (w *Watcher) loop() {
	for {
		select {
		case ev, ok := <-w.fs.Events:
			if !ok {
				return
			}
			rel, _ := filepath.Rel(w.root, ev.Name)
			out := fsEvent{Op: ev.Op.String(), Path: rel}
			w.broadcast(out)
			if ev.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
					_ = w.fs.Add(ev.Name)
				}
			}
		case err, ok := <-w.fs.Errors:
			if !ok {
				return
			}
			log.Printf("watcher: %v", err)
		}
	}
}

func (w *Watcher) broadcast(ev fsEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for ch := range w.clients {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (w *Watcher) subscribe() chan fsEvent {
	ch := make(chan fsEvent, 32)
	w.mu.Lock()
	w.clients[ch] = struct{}{}
	w.mu.Unlock()
	return ch
}

func (w *Watcher) unsubscribe(ch chan fsEvent) {
	w.mu.Lock()
	delete(w.clients, ch)
	w.mu.Unlock()
	close(ch)
}

func (w *Watcher) Close() error { return w.fs.Close() }

func handleEvents(w *Watcher) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(rw, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Printf("events ws accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "bye")

		ch := w.subscribe()
		defer w.unsubscribe(ch)

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		go func() {
			for {
				if _, _, err := conn.Read(ctx); err != nil {
					cancel()
					return
				}
			}
		}()

		ping := time.NewTicker(20 * time.Second)
		defer ping.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ping.C:
				pctx, pcancel := context.WithTimeout(ctx, 10*time.Second)
				if err := conn.Ping(pctx); err != nil {
					pcancel()
					return
				}
				pcancel()
			case ev := <-ch:
				data, _ := json.Marshal(ev)
				if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
					return
				}
			}
		}
	}
}
