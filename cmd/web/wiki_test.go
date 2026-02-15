package main

import "testing"

func TestParseRecipeFromExtract(t *testing.T) {
	wikitext := `{{Infobox
|recipe = Time, 15 + Electronic circuit, 5 + Advanced circuit, 5
}}`

	craftTime, output, err := parseRecipeFromWikitext(wikitext)
	if err != nil {
		t.Fatalf("parseRecipeFromWikitext returned unexpected error: %v", err)
	}
	if craftTime != 15 {
		t.Fatalf("craft time got %f, want 15", craftTime)
	}
	if output != 1 {
		t.Fatalf("output got %f, want 1", output)
	}
}

func TestParseRecipeFromExtract_WithDecimalValues(t *testing.T) {
	wikitext := `{{Infobox
|recipe = Time, 0.5 + Copper plate, 1 = Copper cable, 2
}}`

	craftTime, output, err := parseRecipeFromWikitext(wikitext)
	if err != nil {
		t.Fatalf("parseRecipeFromWikitext returned unexpected error: %v", err)
	}
	if craftTime != 0.5 {
		t.Fatalf("craft time got %f, want 0.5", craftTime)
	}
	if output != 2 {
		t.Fatalf("output got %f, want 2", output)
	}
}

func TestParseRecipeFromExtract_FailsWhenMissingRecipe(t *testing.T) {
	wikitext := `No recipe info here`
	_, _, err := parseRecipeFromWikitext(wikitext)
	if err == nil {
		t.Fatal("expected error when recipe line is missing")
	}
}

func TestParseProducersFromWikitext(t *testing.T) {
	wikitext := `{{Infobox
|producers = Assembling machine + Player
|space-age-producers = Assembling machine + Electromagnetic plant + Player
}}`

	producers := parseProducersFromWikitext(wikitext)
	if len(producers) != 2 {
		t.Fatalf("producer count got %d, want 2", len(producers))
	}
	if producers[0] != "Assembling machine" {
		t.Fatalf("first producer got %q, want %q", producers[0], "Assembling machine")
	}
	if producers[1] != "Electromagnetic plant" {
		t.Fatalf("second producer got %q, want %q", producers[1], "Electromagnetic plant")
	}
}

func TestNormalizeProducerName_WikiMarkup(t *testing.T) {
	got := normalizeProducerName("[[Assembling_machine|Assembling machine]]")
	if got != "Assembling machine" {
		t.Fatalf("normalizeProducerName got %q, want %q", got, "Assembling machine")
	}
}

func TestParseInfoboxNumericValue(t *testing.T) {
	wikitext := `{{Infobox
|crafting-speed = 1.25
|base-productivity = 0
|quality-bonus = 0
}}`

	craftSpeed, ok := parseInfoboxNumericValue(wikitext, []string{"crafting-speed"})
	if !ok {
		t.Fatal("expected crafting-speed to parse")
	}
	if craftSpeed != 1.25 {
		t.Fatalf("craft speed got %f, want 1.25", craftSpeed)
	}
}

func TestMachineInfoboxCandidates_IncludesFallbacks(t *testing.T) {
	candidates := machineInfoboxCandidates("Assembling machine")
	if len(candidates) < 2 {
		t.Fatalf("candidate count got %d, want at least 2", len(candidates))
	}
	if candidates[0] != "Infobox:Assembling_machine" {
		t.Fatalf("first candidate got %q, want %q", candidates[0], "Infobox:Assembling_machine")
	}
}

func TestParseInfoboxNumericValue_ModulesKey(t *testing.T) {
	wikitext := `{{Infobox
|modules = 4
}}`

	moduleSlots, ok := parseInfoboxNumericValue(wikitext, []string{"modules", "module-slots"})
	if !ok {
		t.Fatal("expected modules to parse")
	}
	if moduleSlots != 4 {
		t.Fatalf("module slots got %f, want 4", moduleSlots)
	}
}
