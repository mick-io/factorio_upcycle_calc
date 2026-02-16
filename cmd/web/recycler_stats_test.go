package main

import (
	"net/url"
	"testing"
)

func TestRecycleTimeSourceFromCraftTime(t *testing.T) {
	if got := recycleTimeSourceFromCraftTime(16); got != 1 {
		t.Fatalf("recycle time source got %f, want 1", got)
	}

	if got := recycleTimeSourceFromCraftTime(0); got != 0.0625 {
		t.Fatalf("fallback recycle time source got %f, want 0.0625", got)
	}
}

func TestCalculateRecyclerStats_UsesItemDerivedRecycleTime(t *testing.T) {
	values := url.Values{
		"base_recycler_craft_speed":        {"0.5"},
		"base_recycler_quality_percentage": {"0"},
		"recycler_quality":                 {"normal"},
	}

	stats := calculateRecyclerStats(values, recycleTimeSourceFromCraftTime(16))
	if stats.EffectiveCraftSpeed != 0.5 {
		t.Fatalf("effective craft speed got %f, want 0.5", stats.EffectiveCraftSpeed)
	}
	if stats.EffectiveRecycleTimeSeconds != 2 {
		t.Fatalf("effective recycle cycle time got %f, want 2", stats.EffectiveRecycleTimeSeconds)
	}
}

func TestCalculateRecyclerStats_NormalRecyclerWithFourLegendaryQualityThreeModules(t *testing.T) {
	values := url.Values{
		"base_recycler_craft_speed":        {"0.5"},
		"base_recycler_quality_percentage": {"0"},
		"recycler_quality":                 {"normal"},
		"recycler_module_slot_1":           {"quality-module-3"},
		"recycler_module_slot_1_quality":   {"legendary"},
		"recycler_module_slot_2":           {"quality-module-3"},
		"recycler_module_slot_2_quality":   {"legendary"},
		"recycler_module_slot_3":           {"quality-module-3"},
		"recycler_module_slot_3_quality":   {"legendary"},
		"recycler_module_slot_4":           {"quality-module-3"},
		"recycler_module_slot_4_quality":   {"legendary"},
	}

	stats := calculateRecyclerStats(values, recycleTimeSourceFromCraftTime(16))
	if stats.EffectiveCraftSpeed != 0.4 {
		t.Fatalf("effective craft speed got %f, want 0.4", stats.EffectiveCraftSpeed)
	}
	if stats.EffectiveRecycleTimeSeconds != 2.5 {
		t.Fatalf("effective recycle cycle time got %f, want 2.5", stats.EffectiveRecycleTimeSeconds)
	}
}
