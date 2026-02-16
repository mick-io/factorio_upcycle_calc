package internal

import (
	"errors"
	"math"
)

var (
	errNonPositiveRecyclerCapacity  = errors.New("recycler throughput must be > 0")
	errPlannerInvalidOutputQuantity = errors.New("base output values must be >= 0")
	errNegativeAnchorMachineCount   = errors.New("anchor machine count must be >= 0")
	errNegativeTotalMachineCount    = errors.New("total machine count must be >= 0")
	errNonPositiveAnchorShare       = errors.New("anchor quality has zero expected output share")
)

const (
	qualityTierCount        = int(QualityLegendary) + 1
	recyclerReturnPerItem   = 0.25
	loopSolverTolerance     = 1e-9
	loopSolverMaxIterations = 10000
)

type PlanInput struct {
	TargetQuality            QualityTier
	Machine                  Machine
	Recycler                 Recycler
	BaseOutputPerCraft       float64
	BaseCraftTimeSeconds     float64
	BaseRecycleInputPerCycle float64
	BaseRecycleTimeSeconds   float64
}

type PlanResult struct {
	MachinesByQuality       map[QualityTier]int
	MachineRatioByQuality   map[QualityTier]float64
	TotalMachines           int
	TotalMachinesExact      float64
	RequiredRecyclers       int
	ProducedPerSecond       float64
	RecycleLoadPerSecond    float64
	RecyclerPerSecond       float64
	ExpectedOutputByQuality map[QualityTier]float64
}

// BuildPlanFromTotalMachines computes plan values from a user-provided total machine count.
func BuildPlanFromTotalMachines(input PlanInput, totalMachines int) (PlanResult, error) {
	if totalMachines < 0 {
		return PlanResult{}, errNegativeTotalMachineCount
	}
	if input.BaseOutputPerCraft < 0 || input.BaseRecycleInputPerCycle < 0 {
		return PlanResult{}, errPlannerInvalidOutputQuantity
	}
	return buildPlanWithTotalMachines(input, float64(totalMachines))
}

// BuildPlanFromAnchorQuality computes plan values by anchoring to a user-provided
// machine count for one specific quality tier. All derived counts are rounded up.
func BuildPlanFromAnchorQuality(input PlanInput, anchorQuality QualityTier, anchorMachineCount int) (PlanResult, error) {
	if anchorMachineCount < 0 {
		return PlanResult{}, errNegativeAnchorMachineCount
	}
	if !anchorQuality.IsValid() {
		return PlanResult{}, errInvalidInputQuality
	}
	if input.BaseOutputPerCraft < 0 || input.BaseRecycleInputPerCycle < 0 {
		return PlanResult{}, errPlannerInvalidOutputQuantity
	}

	machineRatioByQuality, err := machineRatioForUpcycling(input)
	if err != nil {
		return PlanResult{}, err
	}

	anchorShare := machineRatioByQuality[anchorQuality]
	if anchorShare <= 0 {
		return PlanResult{}, errNonPositiveAnchorShare
	}

	totalMachines := float64(anchorMachineCount) / anchorShare
	return buildPlanWithTotalMachines(input, totalMachines)
}

func buildPlanWithTotalMachines(input PlanInput, totalMachines float64) (PlanResult, error) {
	if totalMachines < 0 {
		return PlanResult{}, errNegativeAnchorMachineCount
	}

	_, recyclerPerSecond, err := perSecondMachineAndRecycler(input)
	if err != nil {
		return PlanResult{}, err
	}

	machinesByQuality := map[QualityTier]int{
		QualityNormal:    0,
		QualityUncommon:  0,
		QualityRare:      0,
		QualityEpic:      0,
		QualityLegendary: 0,
	}
	machineRatioByQuality := map[QualityTier]float64{
		QualityNormal:    0,
		QualityUncommon:  0,
		QualityRare:      0,
		QualityEpic:      0,
		QualityLegendary: 0,
	}
	expectedOutputByQuality := map[QualityTier]float64{
		QualityNormal:    0,
		QualityUncommon:  0,
		QualityRare:      0,
		QualityEpic:      0,
		QualityLegendary: 0,
	}

	machineRatioByQuality, err = machineRatioForUpcycling(input)
	if err != nil {
		return PlanResult{}, err
	}

	for tier, share := range machineRatioByQuality {
		machinesByQuality[tier] = int(math.Ceil(totalMachines * share))
	}

	producedPerSecond := 0.0
	for tier := QualityNormal; tier <= input.TargetQuality; tier++ {
		share := machineRatioByQuality[tier]
		if share <= 0 {
			continue
		}

		perMachine, err := input.Machine.Craft(
			input.BaseOutputPerCraft,
			input.BaseCraftTimeSeconds,
			1,
			tier,
			input.TargetQuality,
		)
		if err != nil {
			return PlanResult{}, err
		}

		machineCount := totalMachines * share
		producedPerSecond += machineCount * perMachine.TotalOutput
		for outTier, outputPerMachine := range perMachine.OutputByQuality {
			expectedOutputByQuality[outTier] += machineCount * outputPerMachine
		}
	}

	recycleLoadPerSecond := 0.0
	for tier := QualityNormal; tier < input.TargetQuality; tier++ {
		recycleLoadPerSecond += expectedOutputByQuality[tier]
	}

	requiredRecyclers := int(math.Ceil(recycleLoadPerSecond / recyclerPerSecond))
	totalMachinesExact := math.Max(0, totalMachines)
	roundedTotalMachines := int(math.Ceil(math.Max(0, totalMachines)))

	return PlanResult{
		MachinesByQuality:       machinesByQuality,
		MachineRatioByQuality:   machineRatioByQuality,
		TotalMachines:           roundedTotalMachines,
		TotalMachinesExact:      totalMachinesExact,
		RequiredRecyclers:       requiredRecyclers,
		ProducedPerSecond:       producedPerSecond,
		RecycleLoadPerSecond:    recycleLoadPerSecond,
		RecyclerPerSecond:       recyclerPerSecond,
		ExpectedOutputByQuality: expectedOutputByQuality,
	}, nil
}

func machineRatioForUpcycling(input PlanInput) (map[QualityTier]float64, error) {
	if !input.TargetQuality.IsValid() {
		return nil, errInvalidMaxQuality
	}

	ratios := map[QualityTier]float64{
		QualityNormal:    0,
		QualityUncommon:  0,
		QualityRare:      0,
		QualityEpic:      0,
		QualityLegendary: 0,
	}

	if input.TargetQuality == QualityNormal {
		ratios[QualityNormal] = 1
		return ratios, nil
	}

	targetIdx := int(input.TargetQuality)
	assemblerQualityChance := input.Machine.qualityChance()
	recyclerQualityChance := input.Recycler.qualityChance()
	assemblerOutputMultiplier := math.Max(0, 1+(input.Machine.Productivity/100))
	loopScale := assemblerOutputMultiplier * recyclerReturnPerItem

	assemblerProb := [qualityTierCount][qualityTierCount]float64{}
	recyclerProb := [qualityTierCount][qualityTierCount]float64{}
	recycleMask := [qualityTierCount]float64{}

	for inputTier := QualityNormal; inputTier <= input.TargetQuality; inputTier++ {
		distribution := qualityDistribution(inputTier, input.TargetQuality, assemblerQualityChance)
		for outputTier := QualityNormal; outputTier <= input.TargetQuality; outputTier++ {
			assemblerProb[outputTier][inputTier] = distribution[outputTier]
		}
	}

	for recycledTier := QualityNormal; recycledTier < input.TargetQuality; recycledTier++ {
		recycleMask[recycledTier] = 1
		distribution := qualityDistribution(recycledTier, input.TargetQuality, recyclerQualityChance)
		for outputTier := QualityNormal; outputTier <= input.TargetQuality; outputTier++ {
			recyclerProb[outputTier][recycledTier] = distribution[outputTier]
		}
	}

	source := [qualityTierCount]float64{}
	source[QualityNormal] = 1
	materials := source

	for iter := 0; iter < loopSolverMaxIterations; iter++ {
		next := source
		maxDelta := 0.0

		for outTier := 0; outTier <= targetIdx; outTier++ {
			loopContribution := 0.0
			for recycledInputTier := 0; recycledInputTier <= targetIdx; recycledInputTier++ {
				if recycleMask[recycledInputTier] == 0 {
					continue
				}

				recycledInputAmount := 0.0
				for craftInputTier := 0; craftInputTier <= targetIdx; craftInputTier++ {
					recycledInputAmount += assemblerProb[recycledInputTier][craftInputTier] * materials[craftInputTier]
				}

				loopContribution += recyclerProb[outTier][recycledInputTier] * recycledInputAmount
			}

			next[outTier] += loopScale * loopContribution
			delta := math.Abs(next[outTier] - materials[outTier])
			if delta > maxDelta {
				maxDelta = delta
			}
		}

		materials = next
		if maxDelta < loopSolverTolerance {
			break
		}
	}

	total := 0.0
	for tier := 0; tier <= targetIdx; tier++ {
		total += materials[tier]
	}
	if total <= 0 {
		ratios[QualityNormal] = 1
		return ratios, nil
	}

	for tier := 0; tier <= targetIdx; tier++ {
		ratios[QualityTier(tier)] = materials[tier] / total
	}

	return ratios, nil
}

func perSecondMachineAndRecycler(input PlanInput) (CraftResult, float64, error) {
	perMachine, err := input.Machine.Craft(
		input.BaseOutputPerCraft,
		input.BaseCraftTimeSeconds,
		1,
		QualityNormal,
		input.TargetQuality,
	)
	if err != nil {
		return CraftResult{}, 0, err
	}

	if input.BaseRecycleTimeSeconds <= 0 {
		return CraftResult{}, 0, errNonPositiveCraftTime
	}
	if input.BaseRecycleInputPerCycle < 0 {
		return CraftResult{}, 0, errPlannerInvalidOutputQuantity
	}

	// BaseRecycleTimeSeconds is provided as the effective recycle cycle time from UI inputs.
	// Do not multiply by recycler craft speed again here or throughput is double-counted.
	recyclerPerSecond := input.BaseRecycleInputPerCycle / input.BaseRecycleTimeSeconds
	if recyclerPerSecond <= 0 {
		return CraftResult{}, 0, errNonPositiveRecyclerCapacity
	}

	return perMachine, recyclerPerSecond, nil
}

func qualityShareForTier(result CraftResult, tier QualityTier) float64 {
	if result.TotalOutput <= 0 {
		return 0
	}
	return result.OutputByQuality[tier] / result.TotalOutput
}
