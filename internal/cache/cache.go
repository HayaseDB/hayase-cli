package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"

	"github.com/hayasedb/hayase/internal/paths"
)

type Cache struct {
	disabled bool
}

func New(disabled bool) *Cache {
	return &Cache{disabled: disabled}
}

func Clear() error {
	return os.RemoveAll(paths.CacheDir())
}

type Entry[T any] struct {
	CachedAt  time.Time `json:"cached_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Data      T         `json:"data"`
}

func Get[T any](c *Cache, key string) (T, bool) {
	var zero T

	if c.disabled {
		return zero, false
	}

	path, err := paths.CacheFile(key + ".json")
	if err != nil {
		return zero, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return zero, false
	}

	var entry Entry[T]
	if err := json.Unmarshal(data, &entry); err != nil {
		log.Debug("cache unmarshal error", "key", key, "error", err)
		return zero, false
	}

	if time.Now().After(entry.ExpiresAt) {
		log.Debug("cache expired", "key", key, "expired_at", entry.ExpiresAt)
		return zero, false
	}

	log.Debug("cache hit", "key", key)
	return entry.Data, true
}

func Set[T any](c *Cache, key string, data T, ttl time.Duration) {
	path, err := paths.CacheFile(key + ".json")
	if err != nil {
		log.Warn("cache path error", "key", key, "error", err)
		return
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		log.Warn("cache mkdir error", "key", key, "error", err)
		return
	}

	entry := Entry[T]{
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(ttl),
		Data:      data,
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		log.Warn("cache marshal error", "key", key, "error", err)
		return
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, jsonData, 0600); err != nil {
		log.Warn("cache write error", "key", key, "error", err)
		return
	}

	if err := os.Rename(tmpPath, path); err != nil {
		log.Warn("cache rename error", "key", key, "error", err)
		_ = os.Remove(tmpPath)
		return
	}

	log.Debug("cache set", "key", key, "ttl", ttl)
}

func HashKey(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8])
}
