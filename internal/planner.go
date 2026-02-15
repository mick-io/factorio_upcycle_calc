package internal

import (
	"errors"
	"math"
)

var (
	errNonPositiveMachineCount      = errors.New("total crafting machines must be > 0")
	errNonPositiveRecyclerCapacity  = errors.New("recycler throughput must be > 0")
	errPlannerInvalidOutputQuantity = errors.New("base output values must be >= 0")
	errNegativeAnchorMachineCount   = errors.New("anchor machine count must be >= 0")
	errNonPositiveAnchorShare       = errors.New("anchor quality has zero expected output share")
)

type PlanInput struct {
	TargetQuality            QualityTier
	TotalCraftingMachines    int
	Machine                  Machine
	Recycler                 Recycler
	BaseOutputPerCraft       float64
	BaseCraftTimeSeconds     float64
	BaseRecycleInputPerCycle float64
	BaseRecycleTimeSeconds   float64
}

type PlanResult struct {
	MachinesByQuality       map[QualityTier]int
	RequiredRecyclers       int
	ProducedPerSecond       float64
	RecycleLoadPerSecond    float64
	RecyclerPerSecond       float64
	ExpectedOutputByQuality map[QualityTier]float64
}

// BuildPlan computes rounded-up machine allocations per quality tier and
// rounded-up recycler count needed to support produced volume.
func BuildPlan(input PlanInput) (PlanResult, error) {
	if input.TotalCraftingMachines <= 0 {
		return PlanResult{}, errNonPositiveMachineCount
	}
	if input.BaseOutputPerCraft < 0 || input.BaseRecycleInputPerCycle < 0 {
		return PlanResult{}, errPlannerInvalidOutputQuantity
	}

	return buildPlanWithTotalMachines(input, float64(input.TotalCraftingMachines))
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

	perMachine, _, err := perSecondMachineAndRecycler(input)
	if err != nil {
		return PlanResult{}, err
	}

	anchorPerMachineOutput := perMachine.OutputByQuality[anchorQuality]
	anchorShare := 0.0
	if perMachine.TotalOutput > 0 {
		anchorShare = anchorPerMachineOutput / perMachine.TotalOutput
	}
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

	perMachine, recyclerPerSecond, err := perSecondMachineAndRecycler(input)
	if err != nil {
		return PlanResult{}, err
	}

	producedPerSecond := perMachine.TotalOutput * totalMachines
	targetOutputPerSecond := producedPerSecond * qualityShareForTier(perMachine, input.TargetQuality)
	recycleLoadPerSecond := producedPerSecond - targetOutputPerSecond
	if recycleLoadPerSecond < 0 {
		recycleLoadPerSecond = 0
	}

	machinesByQuality := map[QualityTier]int{
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

	for tier, perMachineOutput := range perMachine.OutputByQuality {
		share := 0.0
		if perMachine.TotalOutput > 0 {
			share = perMachineOutput / perMachine.TotalOutput
		}

		machinesByQuality[tier] = int(math.Ceil(totalMachines * share))
		expectedOutputByQuality[tier] = producedPerSecond * share
	}

	requiredRecyclers := int(math.Ceil(recycleLoadPerSecond / recyclerPerSecond))

	return PlanResult{
		MachinesByQuality:       machinesByQuality,
		RequiredRecyclers:       requiredRecyclers,
		ProducedPerSecond:       producedPerSecond,
		RecycleLoadPerSecond:    recycleLoadPerSecond,
		RecyclerPerSecond:       recyclerPerSecond,
		ExpectedOutputByQuality: expectedOutputByQuality,
	}, nil
}

func perSecondMachineAndRecycler(input PlanInput) (CraftResult, float64, error) {
	perMachine, err := input.Machine.Craft(
		Item{},
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

	effectiveRecyclerCraftSpeed := math.Max(0, input.Recycler.CraftSpeed)
	recyclerPerSecond := (effectiveRecyclerCraftSpeed * input.BaseRecycleInputPerCycle) / input.BaseRecycleTimeSeconds
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
