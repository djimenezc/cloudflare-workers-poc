package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	workspace := envOr("WORKSPACE_DIR", "/workspace")
	if err := seedWorkspace(workspace); err != nil {
		log.Fatalf("seed workspace: %v", err)
	}

	watcher, err := newWatcher(workspace)
	if err != nil {
		log.Fatalf("watcher: %v", err)
	}
	defer watcher.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/tree", handleTree(workspace))
	mux.HandleFunc("GET /api/file", handleReadFile(workspace))
	mux.HandleFunc("PUT /api/file", handleWriteFile(workspace))
	mux.HandleFunc("/api/terminal", handleTerminal(workspace))
	mux.HandleFunc("/api/events", handleEvents(watcher))
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.Handle("/preview/", http.StripPrefix("/preview/", http.FileServer(http.Dir(workspace+"/public"))))

	server := &http.Server{
		Addr:    ":8080",
		Handler: withLogging(mux),
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("workspace agent listening on %s (workspace=%s)", server.Addr, workspace)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-stop
	log.Print("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func seedWorkspace(dir string) error {
	if _, err := os.Stat(dir + "/.seeded"); err == nil {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	cmd := exec.Command("cp", "-a", "/opt/seed/.", dir+"/")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return os.WriteFile(dir+"/.seeded", []byte("1"), 0o644)
}

func withLogging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
