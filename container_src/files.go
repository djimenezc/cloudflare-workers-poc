package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Entry struct {
	Name     string  `json:"name"`
	Path     string  `json:"path"`
	Type     string  `json:"type"`
	Children []Entry `json:"children,omitempty"`
}

func handleTree(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tree, err := buildTree(root, "")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tree)
	}
}

func buildTree(root, rel string) (Entry, error) {
	abs := filepath.Join(root, rel)
	info, err := os.Stat(abs)
	if err != nil {
		return Entry{}, err
	}
	name := filepath.Base(rel)
	if rel == "" {
		name = "workspace"
	}
	e := Entry{Name: name, Path: rel, Type: "file"}
	if !info.IsDir() {
		return e, nil
	}
	e.Type = "dir"
	entries, err := os.ReadDir(abs)
	if err != nil {
		return e, err
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})
	for _, child := range entries {
		if strings.HasPrefix(child.Name(), ".") {
			continue
		}
		sub, err := buildTree(root, filepath.Join(rel, child.Name()))
		if err != nil {
			continue
		}
		e.Children = append(e.Children, sub)
	}
	return e, nil
}

func safeJoin(root, rel string) (string, error) {
	cleaned := filepath.Clean("/" + rel)
	abs := filepath.Join(root, cleaned)
	if !strings.HasPrefix(abs, filepath.Clean(root)+string(os.PathSeparator)) && abs != filepath.Clean(root) {
		return "", errors.New("path escapes workspace")
	}
	return abs, nil
}

func handleReadFile(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		if path == "" {
			http.Error(w, "missing path", http.StatusBadRequest)
			return
		}
		abs, err := safeJoin(root, path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		f, err := os.Open(abs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.Copy(w, f)
	}
}

func handleWriteFile(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		if path == "" {
			http.Error(w, "missing path", http.StatusBadRequest)
			return
		}
		abs, err := safeJoin(root, path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := os.WriteFile(abs, body, 0o644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
