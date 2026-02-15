package main

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestMachineDetailsCache_GetOrFetch_CachesAndPersists(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "wiki-machine-details.json")

	cache, err := newMachineDetailsCache(cachePath)
	if err != nil {
		t.Fatalf("newMachineDetailsCache returned unexpected error: %v", err)
	}

	fetchCalls := 0
	fetch := func(machine string) (MachineWikiDetails, error) {
		fetchCalls++
		return MachineWikiDetails{
			Machine:                machine,
			CraftSpeed:             1.25,
			ProductivityPercentage: 0,
			QualityPercentage:      0,
			ModuleSlots:            4,
			SourcePage:             "Infobox:Assembling_machine_3",
			DataParsed:             true,
			ModuleSlotsParsed:      true,
			CacheVersion:           machineDetailsCacheVersion,
		}, nil
	}

	first, err := cache.GetOrFetch("Assembling machine", fetch)
	if err != nil {
		t.Fatalf("GetOrFetch first call returned unexpected error: %v", err)
	}
	if first.CraftSpeed != 1.25 {
		t.Fatalf("first result craft speed got %f, want 1.25", first.CraftSpeed)
	}

	second, err := cache.GetOrFetch("Assembling machine", fetch)
	if err != nil {
		t.Fatalf("GetOrFetch second call returned unexpected error: %v", err)
	}
	if second.ProductivityPercentage != 0 {
		t.Fatalf("second result productivity got %f, want 0", second.ProductivityPercentage)
	}
	if fetchCalls != 1 {
		t.Fatalf("fetch calls got %d, want 1", fetchCalls)
	}

	reloadedCache, err := newMachineDetailsCache(cachePath)
	if err != nil {
		t.Fatalf("reloaded newMachineDetailsCache returned unexpected error: %v", err)
	}

	_, err = reloadedCache.GetOrFetch("Assembling machine", func(machine string) (MachineWikiDetails, error) {
		t.Fatalf("fetch should not be called after cache reload")
		return MachineWikiDetails{}, nil
	})
	if err != nil {
		t.Fatalf("GetOrFetch after reload returned unexpected error: %v", err)
	}
}

func TestNormalizeMachineKey(t *testing.T) {
	got := normalizeMachineKey("  Assembling machine  ")
	if got != "assembling machine" {
		t.Fatalf("normalizeMachineKey got %q, want %q", got, "assembling machine")
	}
}

func TestMachineDetailsCache_GetOrFetch_RefreshesLegacyEntries(t *testing.T) {
	cache := &machineDetailsCache{
		entries: map[string]MachineWikiDetails{
			"assembling machine": {
				Machine:           "Assembling machine",
				CraftSpeed:        1.25,
				SourcePage:        "Infobox:Assembling_machine_3",
				DataParsed:        false,
				ModuleSlotsParsed: false,
				CacheVersion:      0,
			},
		},
	}

	fetchCalls := 0
	_, err := cache.GetOrFetch("Assembling machine", func(machine string) (MachineWikiDetails, error) {
		fetchCalls++
		return MachineWikiDetails{
			Machine:                machine,
			CraftSpeed:             1.25,
			ProductivityPercentage: 0,
			QualityPercentage:      0,
			ModuleSlots:            4,
			SourcePage:             "Infobox:Assembling_machine_3",
			DataParsed:             true,
			ModuleSlotsParsed:      true,
			CacheVersion:           machineDetailsCacheVersion,
		}, nil
	})
	if err != nil {
		t.Fatalf("GetOrFetch returned unexpected error: %v", err)
	}
	if fetchCalls != 1 {
		t.Fatalf("fetch calls got %d, want 1", fetchCalls)
	}
}

func TestMachineDetailsCache_GetOrFetch_LegacyEntryFallsBackWhenFetchFails(t *testing.T) {
	cache := &machineDetailsCache{
		entries: map[string]MachineWikiDetails{
			"assembling machine": {
				Machine:           "Assembling machine",
				CraftSpeed:        1.25,
				SourcePage:        "Infobox:Assembling_machine_3",
				DataParsed:        false,
				ModuleSlotsParsed: false,
				CacheVersion:      0,
			},
		},
	}

	got, err := cache.GetOrFetch("Assembling machine", func(machine string) (MachineWikiDetails, error) {
		return MachineWikiDetails{}, errors.New("wiki unavailable")
	})
	if err != nil {
		t.Fatalf("GetOrFetch returned unexpected error: %v", err)
	}
	if got.Machine != "Assembling machine" {
		t.Fatalf("fallback machine got %q, want %q", got.Machine, "Assembling machine")
	}
}

func TestMachineDetailsCache_GetOrFetch_RefreshesOldCacheVersion(t *testing.T) {
	cache := &machineDetailsCache{
		entries: map[string]MachineWikiDetails{
			"assembling machine": {
				Machine:                "Assembling machine",
				CraftSpeed:             1.25,
				ProductivityPercentage: 0,
				QualityPercentage:      0,
				ModuleSlots:            0,
				SourcePage:             "Infobox:Assembling_machine_3",
				DataParsed:             true,
				ModuleSlotsParsed:      true,
				CacheVersion:           0,
			},
		},
	}

	fetchCalls := 0
	got, err := cache.GetOrFetch("Assembling machine", func(machine string) (MachineWikiDetails, error) {
		fetchCalls++
		return MachineWikiDetails{
			Machine:                machine,
			CraftSpeed:             1.25,
			ProductivityPercentage: 0,
			QualityPercentage:      0,
			ModuleSlots:            4,
			SourcePage:             "Infobox:Assembling_machine_3",
			DataParsed:             true,
			ModuleSlotsParsed:      true,
			CacheVersion:           machineDetailsCacheVersion,
		}, nil
	})
	if err != nil {
		t.Fatalf("GetOrFetch returned unexpected error: %v", err)
	}
	if fetchCalls != 1 {
		t.Fatalf("fetch calls got %d, want 1", fetchCalls)
	}
	if got.ModuleSlots != 4 {
		t.Fatalf("module slots got %d, want 4", got.ModuleSlots)
	}
}
