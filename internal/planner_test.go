package internal

import "testing"

func TestBuildPlanFromTotalMachines_UsesProvidedTotal(t *testing.T) {
	plan, err := BuildPlanFromTotalMachines(PlanInput{
		TargetQuality:            QualityLegendary,
		Machine:                  Machine{Productivity: 0, CraftSpeed: 1, QualityPercentage: 0},
		Recycler:                 Recycler{CraftSpeed: 1, QualityPercentage: 0},
		BaseOutputPerCraft:       1,
		BaseCraftTimeSeconds:     1,
		BaseRecycleInputPerCycle: 1,
		BaseRecycleTimeSeconds:   1,
	}, 20)
	if err != nil {
		t.Fatalf("BuildPlanFromTotalMachines returned unexpected error: %v", err)
	}

	if got := plan.TotalMachines; got != 20 {
		t.Fatalf("total machines got %d, want 20", got)
	}
	if got := plan.MachinesByQuality[QualityNormal]; got != 20 {
		t.Fatalf("normal machines got %d, want 20", got)
	}
}

func TestBuildPlanFromAnchorQuality_ComputesMachineRatios(t *testing.T) {
	plan, err := BuildPlanFromAnchorQuality(PlanInput{
		TargetQuality:            QualityRare,
		Machine:                  Machine{Productivity: 0, CraftSpeed: 1, QualityPercentage: 20},
		Recycler:                 Recycler{CraftSpeed: 1, QualityPercentage: 0},
		BaseOutputPerCraft:       1,
		BaseCraftTimeSeconds:     1,
		BaseRecycleInputPerCycle: 1,
		BaseRecycleTimeSeconds:   1,
	}, QualityNormal, 10)
	if err != nil {
		t.Fatalf("BuildPlanFromAnchorQuality returned unexpected error: %v", err)
	}

	totalRatio := plan.MachineRatioByQuality[QualityNormal] +
		plan.MachineRatioByQuality[QualityUncommon] +
		plan.MachineRatioByQuality[QualityRare]
	assertApproxEqual(t, totalRatio, 1)

	// In an upcycling loop that recycles everything below target quality,
	// losses push demand heavily toward lower-quality crafting.
	if got := plan.MachineRatioByQuality[QualityNormal]; got <= 0.8 {
		t.Fatalf("normal machine ratio got %f, want > 0.8 in loop model", got)
	}
	if got := plan.MachineRatioByQuality[QualityUncommon]; got <= 0 {
		t.Fatalf("uncommon machine ratio got %f, want > 0", got)
	}

	assertApproxEqual(t, plan.MachineRatioByQuality[QualityEpic], 0)
	assertApproxEqual(t, plan.MachineRatioByQuality[QualityLegendary], 0)
}

func TestBuildPlanFromAnchorQuality_RoundsUpRequiredRecyclers(t *testing.T) {
	plan, err := BuildPlanFromAnchorQuality(PlanInput{
		TargetQuality:            QualityLegendary,
		Machine:                  Machine{Productivity: 0, CraftSpeed: 1, QualityPercentage: 0},
		Recycler:                 Recycler{CraftSpeed: 1, QualityPercentage: 0},
		BaseOutputPerCraft:       1,
		BaseCraftTimeSeconds:     1,
		BaseRecycleInputPerCycle: 0.3,
		BaseRecycleTimeSeconds:   1,
	}, QualityNormal, 10)
	if err != nil {
		t.Fatalf("BuildPlanFromAnchorQuality returned unexpected error: %v", err)
	}

	// Produced/s = 10, recycler/s = 0.3 => ceil(33.333...) = 34
	if got := plan.RequiredRecyclers; got != 34 {
		t.Fatalf("required recyclers got %d, want 34", got)
	}
}

func TestBuildPlanFromTotalMachines_DoesNotDoubleCountRecyclerSpeed(t *testing.T) {
	plan, err := BuildPlanFromTotalMachines(PlanInput{
		TargetQuality:            QualityLegendary,
		Machine:                  Machine{Productivity: 0, CraftSpeed: 1, QualityPercentage: 0},
		Recycler:                 Recycler{CraftSpeed: 3, QualityPercentage: 0},
		BaseOutputPerCraft:       1,
		BaseCraftTimeSeconds:     1,
		BaseRecycleInputPerCycle: 1,
		BaseRecycleTimeSeconds:   0.5,
	}, 10)
	if err != nil {
		t.Fatalf("BuildPlanFromTotalMachines returned unexpected error: %v", err)
	}

	// With 10 items/s to recycle and effective recycle cycle time = 0.5s:
	// recycler throughput = 1 / 0.5 = 2 items/s => ceil(10 / 2) = 5 recyclers.
	if got := plan.RequiredRecyclers; got != 5 {
		t.Fatalf("required recyclers got %d, want 5", got)
	}
}

func TestBuildPlanFromAnchorQuality_RoundsAndMatchesAnchor(t *testing.T) {
	plan, err := BuildPlanFromAnchorQuality(
		PlanInput{
			TargetQuality:            QualityLegendary,
			Machine:                  Machine{Productivity: 0, CraftSpeed: 1, QualityPercentage: 10},
			Recycler:                 Recycler{CraftSpeed: 1, QualityPercentage: 0},
			BaseOutputPerCraft:       1,
			BaseCraftTimeSeconds:     1,
			BaseRecycleInputPerCycle: 0.25,
			BaseRecycleTimeSeconds:   1,
		},
		QualityUncommon,
		10,
	)
	if err != nil {
		t.Fatalf("BuildPlanFromAnchorQuality returned unexpected error: %v", err)
	}

	if got := plan.MachinesByQuality[QualityUncommon]; got != 10 {
		t.Fatalf("uncommon machines got %d, want 10", got)
	}
	if got := plan.MachinesByQuality[QualityNormal]; got <= plan.MachinesByQuality[QualityUncommon] {
		t.Fatalf("normal machines got %d, want greater than uncommon machines", got)
	}
	if got := plan.MachinesByQuality[QualityRare]; got <= 0 {
		t.Fatalf("rare machines got %d, want > 0", got)
	}
}

func TestBuildPlanFromAnchorQuality_ErrorsOnZeroShareAnchor(t *testing.T) {
	_, err := BuildPlanFromAnchorQuality(
		PlanInput{
			TargetQuality:            QualityRare,
			Machine:                  Machine{Productivity: 0, CraftSpeed: 1, QualityPercentage: 20},
			Recycler:                 Recycler{CraftSpeed: 1, QualityPercentage: 0},
			BaseOutputPerCraft:       1,
			BaseCraftTimeSeconds:     1,
			BaseRecycleInputPerCycle: 1,
			BaseRecycleTimeSeconds:   1,
		},
		QualityEpic,
		1,
	)
	if err == nil {
		t.Fatal("expected error for anchor quality with zero share")
	}
}

func TestBuildPlanFromAnchorQuality_RecyclerLoadExcludesTargetQualityOutput(t *testing.T) {
	plan, err := BuildPlanFromAnchorQuality(PlanInput{
		TargetQuality:            QualityNormal,
		Machine:                  Machine{Productivity: 0, CraftSpeed: 1, QualityPercentage: 0},
		Recycler:                 Recycler{CraftSpeed: 1, QualityPercentage: 0},
		BaseOutputPerCraft:       1,
		BaseCraftTimeSeconds:     1,
		BaseRecycleInputPerCycle: 1,
		BaseRecycleTimeSeconds:   1,
	}, QualityNormal, 20)
	if err != nil {
		t.Fatalf("BuildPlanFromAnchorQuality returned unexpected error: %v", err)
	}

	if got := plan.RecycleLoadPerSecond; got != 0 {
		t.Fatalf("recycle load got %f, want 0", got)
	}
	if got := plan.RequiredRecyclers; got != 0 {
		t.Fatalf("required recyclers got %d, want 0", got)
	}
}

func TestBuildPlanFromTotalMachines_RecyclerLoadRecyclesOnlyBelowTargetQuality(t *testing.T) {
	plan, err := BuildPlanFromTotalMachines(PlanInput{
		TargetQuality:            QualityRare,
		Machine:                  Machine{Productivity: 0, CraftSpeed: 1, QualityPercentage: 20},
		Recycler:                 Recycler{CraftSpeed: 1, QualityPercentage: 10},
		BaseOutputPerCraft:       1,
		BaseCraftTimeSeconds:     1,
		BaseRecycleInputPerCycle: 1,
		BaseRecycleTimeSeconds:   1,
	}, 50)
	if err != nil {
		t.Fatalf("BuildPlanFromTotalMachines returned unexpected error: %v", err)
	}

	recycled := plan.ExpectedOutputByQuality[QualityNormal] + plan.ExpectedOutputByQuality[QualityUncommon]
	assertApproxEqual(t, plan.RecycleLoadPerSecond, recycled)
}
