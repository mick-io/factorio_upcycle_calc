package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type itemDetailsCache struct {
	mu      sync.RWMutex
	path    string
	entries map[string]ItemWikiDetails
}

type itemDetailsCacheFile struct {
	Entries map[string]ItemWikiDetails `json:"entries"`
}

func newItemDetailsCache(path string) (*itemDetailsCache, error) {
	c := &itemDetailsCache{
		path:    path,
		entries: map[string]ItemWikiDetails{},
	}

	if err := c.load(); err != nil {
		return c, err
	}
	return c, nil
}

func (c *itemDetailsCache) Get(item string) (ItemWikiDetails, bool) {
	key := normalizeItemKey(item)
	if key == "" {
		return ItemWikiDetails{}, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	details, ok := c.entries[key]
	return details, ok
}

func (c *itemDetailsCache) Set(details ItemWikiDetails) error {
	key := normalizeItemKey(details.Item)
	if key == "" {
		return nil
	}

	c.mu.Lock()
	c.entries[key] = details
	c.mu.Unlock()
	return c.persist()
}

func (c *itemDetailsCache) GetOrFetch(item string, fetch func(string) (ItemWikiDetails, error)) (ItemWikiDetails, error) {
	if cached, ok := c.Get(item); ok {
		metricsCollector.itemCacheHits.Add(1)
		if cached.ProducersParsed {
			return cached, nil
		}

		details, err := fetch(item)
		if err != nil {
			return cached, nil
		}
		_ = c.Set(details)
		return details, nil
	}
	metricsCollector.itemCacheMisses.Add(1)

	details, err := fetch(item)
	if err != nil {
		return ItemWikiDetails{}, err
	}
	_ = c.Set(details)
	return details, nil
}

func (c *itemDetailsCache) load() error {
	if c.path == "" {
		return nil
	}

	b, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var data itemDetailsCacheFile
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	if data.Entries == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	for key, value := range data.Entries {
		c.entries[normalizeItemKey(key)] = value
	}
	return nil
}

func (c *itemDetailsCache) persist() error {
	if c.path == "" {
		return nil
	}

	c.mu.RLock()
	data := itemDetailsCacheFile{
		Entries: make(map[string]ItemWikiDetails, len(c.entries)),
	}
	for k, v := range c.entries {
		data.Entries[k] = v
	}
	c.mu.RUnlock()

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return err
	}

	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, c.path)
}

func normalizeItemKey(item string) string {
	return strings.ToLower(strings.TrimSpace(item))
}
