package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const machineDetailsCacheVersion = 1

type machineDetailsCache struct {
	mu      sync.RWMutex
	path    string
	entries map[string]MachineWikiDetails
}

type machineDetailsCacheFile struct {
	Entries map[string]MachineWikiDetails `json:"entries"`
}

func newMachineDetailsCache(path string) (*machineDetailsCache, error) {
	c := &machineDetailsCache{
		path:    path,
		entries: map[string]MachineWikiDetails{},
	}

	if err := c.load(); err != nil {
		return c, err
	}
	return c, nil
}

func (c *machineDetailsCache) Get(machine string) (MachineWikiDetails, bool) {
	key := normalizeMachineKey(machine)
	if key == "" {
		return MachineWikiDetails{}, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	details, ok := c.entries[key]
	return details, ok
}

func (c *machineDetailsCache) Set(details MachineWikiDetails) error {
	key := normalizeMachineKey(details.Machine)
	if key == "" {
		return nil
	}

	c.mu.Lock()
	c.entries[key] = details
	c.mu.Unlock()
	return c.persist()
}

func (c *machineDetailsCache) GetOrFetch(machine string, fetch func(string) (MachineWikiDetails, error)) (MachineWikiDetails, error) {
	if cached, ok := c.Get(machine); ok {
		metricsCollector.machineCacheHits.Add(1)
		if cached.DataParsed && cached.ModuleSlotsParsed && cached.CacheVersion == machineDetailsCacheVersion {
			return cached, nil
		}

		details, err := fetch(machine)
		if err != nil {
			return cached, nil
		}
		_ = c.Set(details)
		return details, nil
	}
	metricsCollector.machineCacheMisses.Add(1)

	details, err := fetch(machine)
	if err != nil {
		return MachineWikiDetails{}, err
	}
	_ = c.Set(details)
	return details, nil
}

func (c *machineDetailsCache) load() error {
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

	var data machineDetailsCacheFile
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	if data.Entries == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	for key, value := range data.Entries {
		c.entries[normalizeMachineKey(key)] = value
	}
	return nil
}

func (c *machineDetailsCache) persist() error {
	if c.path == "" {
		return nil
	}

	c.mu.RLock()
	data := machineDetailsCacheFile{
		Entries: make(map[string]MachineWikiDetails, len(c.entries)),
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

func normalizeMachineKey(machine string) string {
	return strings.ToLower(strings.TrimSpace(machine))
}
