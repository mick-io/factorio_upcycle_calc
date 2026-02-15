package internal

import "math"

// Recycler holds the core recycling modifiers for a recycler machine.
type Recycler struct {
	CraftSpeed        float64
	QualityPercentage float64
}

// RecycleResult is the expected-value output for a recycle interval.
type RecycleResult struct {
	Item            Item
	TotalOutput     float64
	OutputByQuality map[QualityTier]float64
}

// Recycle computes expected recycled output over a time interval.
// Craft speed increases recycle operations performed, and quality percentage
// distributes output across quality tiers.
func (r Recycler) Recycle(
	item Item,
	baseOutputPerCycle float64,
	baseRecycleTimeSeconds float64,
	durationSeconds float64,
	inputQuality QualityTier,
	maxUnlockedQuality QualityTier,
) (RecycleResult, error) {
	if baseOutputPerCycle < 0 {
		return RecycleResult{}, errNegativeBaseOutput
	}
	if baseRecycleTimeSeconds <= 0 {
		return RecycleResult{}, errNonPositiveCraftTime
	}
	if durationSeconds < 0 {
		return RecycleResult{}, errNegativeDuration
	}
	if !inputQuality.IsValid() {
		return RecycleResult{}, errInvalidInputQuality
	}
	if !maxUnlockedQuality.IsValid() {
		return RecycleResult{}, errInvalidMaxQuality
	}
	if maxUnlockedQuality < inputQuality {
		return RecycleResult{}, errMaxBelowInputQuality
	}

	effectiveCraftSpeed := math.Max(0, r.CraftSpeed)
	cyclesPerformed := (durationSeconds / baseRecycleTimeSeconds) * effectiveCraftSpeed
	totalOutput := cyclesPerformed * baseOutputPerCycle

	qualityShares := qualityDistribution(inputQuality, maxUnlockedQuality, r.qualityChance())
	outputByQuality := make(map[QualityTier]float64, len(qualityShares))
	for tier, share := range qualityShares {
		outputByQuality[tier] = totalOutput * share
	}

	return RecycleResult{
		Item:            item,
		TotalOutput:     totalOutput,
		OutputByQuality: outputByQuality,
	}, nil
}

func (r Recycler) qualityChance() float64 {
	qualityChance := r.QualityPercentage / 100
	if qualityChance < 0 {
		return 0
	}
	if qualityChance > 1 {
		return 1
	}
	return qualityChance
}
