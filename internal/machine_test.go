package internal

import (
	"math"
	"testing"
)

func TestMachineCraft_DistributesAcrossAllQualities(t *testing.T) {
	m := Machine{
		Productivity:      0,
		CraftSpeed:        1,
		QualityPercentage: 10,
	}

	result, err := m.Craft(Item{}, 1, 1, 1, QualityNormal, QualityLegendary)
	if err != nil {
		t.Fatalf("Craft returned unexpected error: %v", err)
	}

	assertApproxEqual(t, result.TotalOutput, 1)
	assertApproxEqual(t, result.OutputByQuality[QualityNormal], 0.9)
	assertApproxEqual(t, result.OutputByQuality[QualityUncommon], 0.09)
	assertApproxEqual(t, result.OutputByQuality[QualityRare], 0.009)
	assertApproxEqual(t, result.OutputByQuality[QualityEpic], 0.0009)
	assertApproxEqual(t, result.OutputByQuality[QualityLegendary], 0.0001)
}

func TestMachineCraft_UsesCraftSpeedAndProductivity(t *testing.T) {
	m := Machine{
		Productivity:      50,
		CraftSpeed:        2,
		QualityPercentage: 0,
	}

	result, err := m.Craft(Item{}, 2, 4, 20, QualityNormal, QualityLegendary)
	if err != nil {
		t.Fatalf("Craft returned unexpected error: %v", err)
	}

	assertApproxEqual(t, result.TotalOutput, 30)
	assertApproxEqual(t, result.OutputByQuality[QualityNormal], 30)
}

func TestMachineCraft_FoldsProbabilityIntoMaxUnlockedTier(t *testing.T) {
	m := Machine{
		Productivity:      0,
		CraftSpeed:        1,
		QualityPercentage: 20,
	}

	result, err := m.Craft(Item{}, 1, 1, 1, QualityNormal, QualityRare)
	if err != nil {
		t.Fatalf("Craft returned unexpected error: %v", err)
	}

	assertApproxEqual(t, result.OutputByQuality[QualityNormal], 0.8)
	assertApproxEqual(t, result.OutputByQuality[QualityUncommon], 0.18)
	assertApproxEqual(t, result.OutputByQuality[QualityRare], 0.02)
}

func TestMachineCraft_StartsFromInputQuality(t *testing.T) {
	m := Machine{
		Productivity:      0,
		CraftSpeed:        1,
		QualityPercentage: 30,
	}

	result, err := m.Craft(Item{}, 1, 1, 1, QualityRare, QualityLegendary)
	if err != nil {
		t.Fatalf("Craft returned unexpected error: %v", err)
	}

	assertApproxEqual(t, result.OutputByQuality[QualityRare], 0.7)
	assertApproxEqual(t, result.OutputByQuality[QualityEpic], 0.27)
	assertApproxEqual(t, result.OutputByQuality[QualityLegendary], 0.03)
}

func TestMachineCraft_ValidatesQualityRange(t *testing.T) {
	m := Machine{
		Productivity:      0,
		CraftSpeed:        1,
		QualityPercentage: 10,
	}

	_, err := m.Craft(Item{}, 1, 1, 1, QualityEpic, QualityRare)
	if err == nil {
		t.Fatal("expected error when max unlocked quality is below input quality")
	}
}

func assertApproxEqual(t *testing.T, got float64, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %f, want %f", got, want)
	}
}
