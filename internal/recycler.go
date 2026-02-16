package internal

import "math"

// Recycler holds the core recycling modifiers used by planning calculations.
type Recycler struct {
	CraftSpeed        float64
	QualityPercentage float64
}

func (r Recycler) qualityChance() float64 {
	qualityChance := r.QualityPercentage / 100
	return math.Min(1, math.Max(0, qualityChance))
}
