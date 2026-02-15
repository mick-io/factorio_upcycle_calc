package internal

import "testing"

func TestRecyclerRecycle_DistributesAcrossAllQualities(t *testing.T) {
	r := Recycler{
		CraftSpeed:        1,
		QualityPercentage: 10,
	}

	result, err := r.Recycle(Item{}, 1, 1, 1, QualityNormal, QualityLegendary)
	if err != nil {
		t.Fatalf("Recycle returned unexpected error: %v", err)
	}

	assertApproxEqual(t, result.TotalOutput, 1)
	assertApproxEqual(t, result.OutputByQuality[QualityNormal], 0.9)
	assertApproxEqual(t, result.OutputByQuality[QualityUncommon], 0.09)
	assertApproxEqual(t, result.OutputByQuality[QualityRare], 0.009)
	assertApproxEqual(t, result.OutputByQuality[QualityEpic], 0.0009)
	assertApproxEqual(t, result.OutputByQuality[QualityLegendary], 0.0001)
}

func TestRecyclerRecycle_UsesCraftSpeed(t *testing.T) {
	r := Recycler{
		CraftSpeed:        2.5,
		QualityPercentage: 0,
	}

	result, err := r.Recycle(Item{}, 3, 2, 8, QualityNormal, QualityLegendary)
	if err != nil {
		t.Fatalf("Recycle returned unexpected error: %v", err)
	}

	// (8 / 2) * 2.5 * 3 = 30
	assertApproxEqual(t, result.TotalOutput, 30)
	assertApproxEqual(t, result.OutputByQuality[QualityNormal], 30)
}

func TestRecyclerRecycle_ValidatesQualityRange(t *testing.T) {
	r := Recycler{
		CraftSpeed:        1,
		QualityPercentage: 15,
	}

	_, err := r.Recycle(Item{}, 1, 1, 1, QualityLegendary, QualityEpic)
	if err == nil {
		t.Fatal("expected error when max unlocked quality is below input quality")
	}
}
