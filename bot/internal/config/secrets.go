package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadSecrets walks dir looking for subdirectories — one per Secret mounted by
// the Helm chart. For each subdir, reads the file named `key` and indexes its
// content (the webhook URL) by subdir name.
//
// Layout produced by the chart:
//
//	dir/
//	  deploy-notify-bot-teams/
//	    webhook-url   ← contains the URL
//	  another-secret/
//	    webhook-url
func LoadSecrets(dir, key string) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read secrets dir %s: %w", dir, err)
	}
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name(), key)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		url := strings.TrimSpace(string(raw))
		if url == "" {
			return nil, fmt.Errorf("secret %s/%s is empty", e.Name(), key)
		}
		out[e.Name()] = url
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no secrets found in %s", dir)
	}
	return out, nil
}
