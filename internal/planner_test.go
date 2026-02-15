package internal

import "testing"

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
	if got := plan.MachinesByQuality[QualityNormal]; got != 100 {
		t.Fatalf("normal machines got %d, want 100", got)
	}
	if got := plan.MachinesByQuality[QualityRare]; got != 1 {
		t.Fatalf("rare machines got %d, want 1", got)
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
