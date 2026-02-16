package main

import (
	"math"
	"testing"
)

func TestScaledModuleBonus_QualityModuleRoundsToTenths(t *testing.T) {
	got := scaledModuleBonus("quality-module-3", 2.5, 2.5)
	if got != 6.2 {
		t.Fatalf("scaled quality bonus got %f, want 6.2", got)
	}
}

func TestScaledModuleBonus_ProductivityModuleRoundsToOnes(t *testing.T) {
	got := scaledModuleBonus("productivity-module-2", 6, 1.6)
	if got != 9 {
		t.Fatalf("scaled productivity bonus got %f, want 9", got)
	}
}

func TestScaledModuleBonus_SpeedModuleRoundsToOnes(t *testing.T) {
	got := scaledModuleBonus("speed-module", 20, 1.9)
	if got != 38 {
		t.Fatalf("scaled speed bonus got %f, want 38", got)
	}
}

func TestRecyclerLegendaryQualityModulesMatchInGameExample(t *testing.T) {
	baseCraftSpeed := 0.5
	totalSpeedPercentModifier := 0.0
	totalQualityBonus := 0.0

	for i := 0; i < 4; i++ {
		effects := moduleEffectsByID["quality-module-3"]
		totalSpeedPercentModifier += scaledModuleBonus("quality-module-3", effects.SpeedBonus, 2.5)
		totalSpeedPercentModifier += effects.SpeedPenalty
		totalQualityBonus += scaledModuleBonus("quality-module-3", effects.QualityBonus, 2.5)
		totalQualityBonus += effects.QualityPenalty
	}

	speedMultiplier := math.Max(0.2, 1+(totalSpeedPercentModifier/100))
	effectiveCraftSpeed := baseCraftSpeed * speedMultiplier
	if effectiveCraftSpeed != 0.4 {
		t.Fatalf("effective recycler craft speed got %f, want 0.4", effectiveCraftSpeed)
	}
	if totalQualityBonus != 24.8 {
		t.Fatalf("effective recycler quality bonus got %f, want 24.8", totalQualityBonus)
	}
}
