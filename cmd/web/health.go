package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}

func readyzHandler(cacheDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := ensureWritableDir(cacheDir); err != nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprintf(w, "not ready: %s", err.Error())
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ready"))
	}
}

func ensureWritableDir(dirPath string) error {
	dirPath = strings.TrimSpace(dirPath)
	if dirPath == "" {
		return fmt.Errorf("writable dir is required")
	}

	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dirPath, err)
	}

	file, err := os.CreateTemp(dirPath, ".ready-*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dirPath, err)
	}
	name := file.Name()
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temp file %s: %w", filepath.Base(name), err)
	}
	if err := os.Remove(name); err != nil {
		return fmt.Errorf("remove temp file %s: %w", filepath.Base(name), err)
	}
	return nil
}
