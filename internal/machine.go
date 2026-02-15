package internal

import (
	"errors"
	"math"
)

var (
	errNegativeBaseOutput   = errors.New("base output per craft must be >= 0")
	errNonPositiveCraftTime = errors.New("base craft time seconds must be > 0")
	errNegativeDuration     = errors.New("duration seconds must be >= 0")
	errInvalidInputQuality  = errors.New("input quality is invalid")
	errInvalidMaxQuality    = errors.New("max unlocked quality is invalid")
	errMaxBelowInputQuality = errors.New("max unlocked quality cannot be below input quality")
)

type QualityTier uint8

const (
	QualityNormal QualityTier = iota
	QualityUncommon
	QualityRare
	QualityEpic
	QualityLegendary
)

func (q QualityTier) IsValid() bool {
	return q <= QualityLegendary
}

func (q QualityTier) String() string {
	switch q {
	case QualityNormal:
		return "Normal"
	case QualityUncommon:
		return "Uncommon"
	case QualityRare:
		return "Rare"
	case QualityEpic:
		return "Epic"
	case QualityLegendary:
		return "Legendary"
	default:
		return "Unknown"
	}
}

// Machine holds the core crafting modifiers for a production machine.
type Machine struct {
	Productivity      float64
	CraftSpeed        float64
	QualityPercentage float64
}

// CraftResult is the expected-value output for a crafting interval.
type CraftResult struct {
	Item            Item
	TotalOutput     float64
	OutputByQuality map[QualityTier]float64
}

// Craft computes expected output over a time interval.
// Productivity increases output quantity, craft speed increases crafts performed,
// and quality percentage distributes output across quality tiers.
func (m Machine) Craft(
	item Item,
	baseOutputPerCraft float64,
	baseCraftTimeSeconds float64,
	durationSeconds float64,
	inputQuality QualityTier,
	maxUnlockedQuality QualityTier,
) (CraftResult, error) {
	if baseOutputPerCraft < 0 {
		return CraftResult{}, errNegativeBaseOutput
	}
	if baseCraftTimeSeconds <= 0 {
		return CraftResult{}, errNonPositiveCraftTime
	}
	if durationSeconds < 0 {
		return CraftResult{}, errNegativeDuration
	}
	if !inputQuality.IsValid() {
		return CraftResult{}, errInvalidInputQuality
	}
	if !maxUnlockedQuality.IsValid() {
		return CraftResult{}, errInvalidMaxQuality
	}
	if maxUnlockedQuality < inputQuality {
		return CraftResult{}, errMaxBelowInputQuality
	}

	effectiveCraftSpeed := math.Max(0, m.CraftSpeed)
	craftsPerformed := (durationSeconds / baseCraftTimeSeconds) * effectiveCraftSpeed
	baseOutput := craftsPerformed * baseOutputPerCraft

	productivityMultiplier := math.Max(0, 1+(m.Productivity/100))
	totalOutput := baseOutput * productivityMultiplier

	qualityShares := qualityDistribution(inputQuality, maxUnlockedQuality, m.qualityChance())
	outputByQuality := make(map[QualityTier]float64, len(qualityShares))
	for tier, share := range qualityShares {
		outputByQuality[tier] = totalOutput * share
	}

	return CraftResult{
		Item:            item,
		TotalOutput:     totalOutput,
		OutputByQuality: outputByQuality,
	}, nil
}

func (m Machine) qualityChance() float64 {
	qualityChance := m.QualityPercentage / 100
	if qualityChance < 0 {
		return 0
	}
	if qualityChance > 1 {
		return 1
	}
	return qualityChance
}

func qualityDistribution(inputQuality QualityTier, maxUnlockedQuality QualityTier, qualityChance float64) map[QualityTier]float64 {
	if maxUnlockedQuality == inputQuality || qualityChance == 0 {
		return map[QualityTier]float64{inputQuality: 1}
	}

	distribution := map[QualityTier]float64{
		inputQuality: 1 - qualityChance,
	}

	jumps := int(maxUnlockedQuality - inputQuality)
	for jump := 1; jump <= jumps; jump++ {
		tier := inputQuality + QualityTier(jump)
		if jump == jumps {
			distribution[tier] = qualityChance * math.Pow(0.1, float64(jump-1))
			continue
		}
		distribution[tier] = 0.9 * qualityChance * math.Pow(0.1, float64(jump-1))
	}

	return distribution
}
