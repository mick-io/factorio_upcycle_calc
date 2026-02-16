package main

const (
	minMachineProductivityPct = 0.0
	maxMachineProductivityPct = 1000.0
	minMachineCraftSpeed      = 0.001
	maxMachineCraftSpeed      = 100.0
	minMachineQualityPct      = 0.0
	maxMachineQualityPct      = 100.0

	minBaseOutputPerCraft = 0.0
	maxBaseOutputPerCraft = 10000.0
	minBaseCraftTimeSec   = 0.001
	maxBaseCraftTimeSec   = 3600.0

	minRecyclerCraftSpeed = 0.001
	maxRecyclerCraftSpeed = 100.0
	minRecyclerQualityPct = 0.0
	maxRecyclerQualityPct = 100.0

	minRecycleInputPerCycle = 0.0
	maxRecycleInputPerCycle = 10000.0
	minRecycleTimeSec       = 0.001
	maxRecycleTimeSec       = 3600.0

	minMachineCount = 0
	maxMachineCount = 1000000
)

func clampFloat(value float64, min float64, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func clampInt(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
