package main

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestItemDetailsCache_GetOrFetch_CachesAndPersists(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "wiki-item-details.json")

	cache, err := newItemDetailsCache(cachePath)
	if err != nil {
		t.Fatalf("newItemDetailsCache returned unexpected error: %v", err)
	}

	fetchCalls := 0
	fetch := func(item string) (ItemWikiDetails, error) {
		fetchCalls++
		return ItemWikiDetails{
			Item:                 item,
			BaseCraftTimeSeconds: 6,
			BaseOutputPerCraft:   1,
			Producers:            []string{"Assembling machine"},
			ProducersParsed:      true,
			SourcePage:           "Infobox:Advanced_circuit",
		}, nil
	}

	first, err := cache.GetOrFetch("advanced-circuit", fetch)
	if err != nil {
		t.Fatalf("GetOrFetch first call returned unexpected error: %v", err)
	}
	if first.BaseCraftTimeSeconds != 6 {
		t.Fatalf("first result craft time got %f, want 6", first.BaseCraftTimeSeconds)
	}

	second, err := cache.GetOrFetch("advanced-circuit", fetch)
	if err != nil {
		t.Fatalf("GetOrFetch second call returned unexpected error: %v", err)
	}
	if second.BaseOutputPerCraft != 1 {
		t.Fatalf("second result output got %f, want 1", second.BaseOutputPerCraft)
	}
	if fetchCalls != 1 {
		t.Fatalf("fetch calls got %d, want 1", fetchCalls)
	}

	reloadedCache, err := newItemDetailsCache(cachePath)
	if err != nil {
		t.Fatalf("reloaded newItemDetailsCache returned unexpected error: %v", err)
	}

	_, err = reloadedCache.GetOrFetch("advanced-circuit", func(item string) (ItemWikiDetails, error) {
		t.Fatalf("fetch should not be called after cache reload")
		return ItemWikiDetails{}, nil
	})
	if err != nil {
		t.Fatalf("GetOrFetch after reload returned unexpected error: %v", err)
	}
}

func TestNormalizeItemKey(t *testing.T) {
	got := normalizeItemKey("  Advanced-Circuit  ")
	if got != "advanced-circuit" {
		t.Fatalf("normalizeItemKey got %q, want %q", got, "advanced-circuit")
	}
}

func TestItemDetailsCache_GetOrFetch_RefreshesLegacyEntries(t *testing.T) {
	cache := &itemDetailsCache{
		entries: map[string]ItemWikiDetails{
			"advanced-circuit": {
				Item:                 "advanced-circuit",
				BaseCraftTimeSeconds: 6,
				BaseOutputPerCraft:   1,
				SourcePage:           "Infobox:Advanced_circuit",
				// Simulates pre-producers cache entries.
				ProducersParsed: false,
			},
		},
	}

	fetchCalls := 0
	_, err := cache.GetOrFetch("advanced-circuit", func(item string) (ItemWikiDetails, error) {
		fetchCalls++
		return ItemWikiDetails{
			Item:                 item,
			BaseCraftTimeSeconds: 6,
			BaseOutputPerCraft:   1,
			Producers:            []string{"Assembling machine"},
			ProducersParsed:      true,
			SourcePage:           "Infobox:Advanced_circuit",
		}, nil
	})
	if err != nil {
		t.Fatalf("GetOrFetch returned unexpected error: %v", err)
	}
	if fetchCalls != 1 {
		t.Fatalf("fetch calls got %d, want 1", fetchCalls)
	}
}

func TestItemDetailsCache_GetOrFetch_LegacyEntryFallsBackWhenFetchFails(t *testing.T) {
	cache := &itemDetailsCache{
		entries: map[string]ItemWikiDetails{
			"advanced-circuit": {
				Item:                 "advanced-circuit",
				BaseCraftTimeSeconds: 6,
				BaseOutputPerCraft:   1,
				SourcePage:           "Infobox:Advanced_circuit",
				ProducersParsed:      false,
			},
		},
	}

	got, err := cache.GetOrFetch("advanced-circuit", func(item string) (ItemWikiDetails, error) {
		return ItemWikiDetails{}, errors.New("wiki unavailable")
	})
	if err != nil {
		t.Fatalf("GetOrFetch returned unexpected error: %v", err)
	}
	if got.Item != "advanced-circuit" {
		t.Fatalf("fallback item got %q, want %q", got.Item, "advanced-circuit")
	}
}
