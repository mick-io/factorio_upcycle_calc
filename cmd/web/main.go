package main

import (
	"bufio"
	"context"
	"fmt"
	"html"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mick-io/factorio_upcycle_calc/internal"
)

type moduleEffect struct {
	SpeedBonus        float64
	SpeedPenalty      float64
	ProductivityBonus float64
	QualityBonus      float64
	QualityPenalty    float64
}

const (
	defaultRecipeCraftTimeSeconds = 1.0
	recycleTimeDivisor            = 16.0
)

var qualityMultiplierByTierKey = map[string]float64{
	"normal":    1.0,
	"uncommon":  1.3,
	"rare":      1.6,
	"epic":      1.9,
	"legendary": 2.5,
}

var moduleEffectsByID = map[string]moduleEffect{
	"speed-module": {
		SpeedBonus:     20,
		QualityPenalty: -1,
	},
	"speed-module-2": {
		SpeedBonus:     30,
		QualityPenalty: -1.5,
	},
	"speed-module-3": {
		SpeedBonus:     50,
		QualityPenalty: -2.5,
	},
	"productivity-module": {
		ProductivityBonus: 4,
		SpeedPenalty:      -5,
	},
	"productivity-module-2": {
		ProductivityBonus: 6,
		SpeedPenalty:      -10,
	},
	"productivity-module-3": {
		ProductivityBonus: 10,
		SpeedPenalty:      -15,
	},
	"quality-module": {
		QualityBonus: 1,
		SpeedPenalty: -5,
	},
	"quality-module-2": {
		QualityBonus: 2,
		SpeedPenalty: -5,
	},
	"quality-module-3": {
		QualityBonus: 2.5,
		SpeedPenalty: -5,
	},
}

func isQualityModule(moduleID string) bool {
	return strings.HasPrefix(strings.TrimSpace(strings.ToLower(moduleID)), "quality-module")
}

func roundScaledModuleBonus(moduleID string, value float64) float64 {
	if value <= 0 {
		return 0
	}
	// Factorio rounds module effects by type:
	// - Quality module bonuses: round down to nearest 0.1%.
	// - Non-quality module positive effects: round down to nearest 1%.
	if isQualityModule(moduleID) {
		return math.Floor(value*10) / 10
	}
	return math.Floor(value)
}

func scaledModuleBonus(moduleID string, baseBonus float64, qualityMultiplier float64) float64 {
	if baseBonus <= 0 || qualityMultiplier <= 0 {
		return 0
	}
	return roundScaledModuleBonus(moduleID, baseBonus*qualityMultiplier)
}

func main() {
	recyclableItems, err := loadRecyclableItems("docs/recyclable-items.txt")
	if err != nil {
		log.Printf("warning: failed to load recyclable items: %v", err)
		recyclableItems = []string{}
	}

	wikiDetailsCache, err := newItemDetailsCache("docs/cache/wiki-item-details.json")
	if err != nil {
		log.Printf("warning: failed to load wiki item-details cache: %v", err)
	}

	machineDetailsCache, err := newMachineDetailsCache("docs/cache/wiki-machine-details.json")
	if err != nil {
		log.Printf("warning: failed to load wiki machine-details cache: %v", err)
	}

	mux := http.NewServeMux()
	cacheDir := "docs/cache"
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/index.html")
	})
	mux.HandleFunc("/healthz", healthzHandler)
	mux.HandleFunc("/readyz", readyzHandler(cacheDir))
	mux.HandleFunc("/partials/recyclable-items", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		var b strings.Builder
		for _, item := range recyclableItems {
			_, _ = fmt.Fprintf(&b, `<option value="%s"></option>`, html.EscapeString(item))
		}
		b.WriteString(renderItemStatus("Loaded recyclable items", "is-success", true))
		_, _ = w.Write([]byte(b.String()))
	})
	mux.HandleFunc("/partials/item-details", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		item := strings.TrimSpace(r.URL.Query().Get("upcycle_item"))
		if item == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(renderItemStatus("Enter an item to load recipe data", "is-primary", false)))
			return
		}

		itemDetails, err := wikiDetailsCache.GetOrFetch(item, fetchItemDetailsFromWiki)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(renderItemStatus(
				fmt.Sprintf("Could not load wiki data for %s", item),
				"is-error",
				false,
			)))
			return
		}

		selectedMachine := selectMachine(itemDetails.Producers, r.URL.Query().Get("craft_machine"))
		maxUnlocked := parseMaxUnlockedQuality(r.URL.Query().Get("max_quality_unlocked"))
		machineDetails, machineErr := resolveMachineDetails(selectedMachine, machineDetailsCache)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		var b strings.Builder

		if machineErr != nil {
			b.WriteString(renderItemStatus(
				fmt.Sprintf("Loaded recipe data for %s. Could not load machine data for %s", itemDetails.Item, selectedMachine),
				"is-error",
				false,
			))
		} else {
			b.WriteString(renderItemStatus(
				fmt.Sprintf("Loaded wiki recipe data for %s", itemDetails.Item),
				"is-success",
				false,
			))
		}

		b.WriteString(renderRecipeFields(itemDetails.BaseOutputPerCraft, itemDetails.BaseCraftTimeSeconds, true))
		baseRecycleTimeSource := recycleTimeSourceFromCraftTime(itemDetails.BaseCraftTimeSeconds)
		b.WriteString(renderRecycleTimeSourceField(baseRecycleTimeSource, true))

		recyclerStats := calculateRecyclerStats(r.URL.Query(), baseRecycleTimeSource)
		b.WriteString(renderRecyclerStatFields(
			recyclerStats.EffectiveCraftSpeed,
			recyclerStats.EffectiveQuality,
			recyclerStats.EffectiveRecycleTimeSeconds,
			recyclerStats.BaseCraftSpeed,
			recyclerStats.BaseQuality,
			recyclerStats.BaseRecycleTimeSeconds,
			true,
		))
		b.WriteString(renderCraftMachineField(itemDetails.Producers, selectedMachine, true))
		b.WriteString(renderMachineModuleField(machineDetails.ModuleSlots, selectedMachine != "", maxUnlocked, true))
		b.WriteString(renderMachineStatFields(
			machineDetails.ProductivityPercentage,
			machineDetails.CraftSpeed,
			machineDetails.QualityPercentage,
			machineDetails.ProductivityPercentage,
			machineDetails.CraftSpeed,
			machineDetails.QualityPercentage,
			true,
		))
		_, _ = w.Write([]byte(b.String()))
	})
	mux.HandleFunc("/partials/machine-details", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		machine := strings.TrimSpace(r.URL.Query().Get("craft_machine"))
		maxUnlocked := parseMaxUnlockedQuality(r.URL.Query().Get("max_quality_unlocked"))

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if machine == "" {
			var b strings.Builder
			b.WriteString(renderItemStatus("Select a crafting machine to load details", "is-primary", false))
			b.WriteString(renderMachineModuleField(0, false, maxUnlocked, true))
			b.WriteString(renderMachineStatFields(0, 1, 0, 0, 1, 0, true))
			_, _ = w.Write([]byte(b.String()))
			return
		}

		machineDetails, err := resolveMachineDetails(machine, machineDetailsCache)
		var b strings.Builder
		if err != nil {
			b.WriteString(renderItemStatus(
				fmt.Sprintf("Could not load machine data for %s", machine),
				"is-error",
				false,
			))
		} else {
			b.WriteString(renderItemStatus(
				fmt.Sprintf("Loaded wiki machine data for %s", machineDetails.Machine),
				"is-success",
				false,
			))
		}

		b.WriteString(renderMachineModuleField(machineDetails.ModuleSlots, true, maxUnlocked, true))
		b.WriteString(renderMachineStatFields(
			machineDetails.ProductivityPercentage,
			machineDetails.CraftSpeed,
			machineDetails.QualityPercentage,
			machineDetails.ProductivityPercentage,
			machineDetails.CraftSpeed,
			machineDetails.QualityPercentage,
			true,
		))
		_, _ = w.Write([]byte(b.String()))
	})
	mux.HandleFunc("/partials/module-stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		machine := strings.TrimSpace(r.URL.Query().Get("craft_machine"))
		machineDetails, _ := resolveMachineDetails(machine, machineDetailsCache)
		baseProductivity := machineDetails.ProductivityPercentage
		baseCraftSpeed := machineDetails.CraftSpeed
		baseQuality := machineDetails.QualityPercentage

		machineQuality := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("machine_quality")))
		machineQualityMultiplier := qualityMultiplierByTierKey[machineQuality]
		if machineQualityMultiplier <= 0 {
			machineQualityMultiplier = 1
		}

		totalSpeedPercentModifier := 0.0
		totalProductivityBonus := 0.0
		totalQualityBonus := 0.0

		for i := 1; i <= machineDetails.ModuleSlots; i++ {
			moduleID := strings.TrimSpace(r.URL.Query().Get(fmt.Sprintf("machine_module_slot_%d", i)))
			if moduleID == "" {
				continue
			}

			effects, ok := moduleEffectsByID[moduleID]
			if !ok {
				continue
			}

			moduleQuality := strings.ToLower(strings.TrimSpace(r.URL.Query().Get(fmt.Sprintf("machine_module_slot_%d_quality", i))))
			moduleQualityMultiplier := qualityMultiplierByTierKey[moduleQuality]
			if moduleQualityMultiplier <= 0 {
				moduleQualityMultiplier = 1
			}

			totalSpeedPercentModifier += scaledModuleBonus(moduleID, effects.SpeedBonus, moduleQualityMultiplier)
			totalSpeedPercentModifier += effects.SpeedPenalty
			totalProductivityBonus += scaledModuleBonus(moduleID, effects.ProductivityBonus, moduleQualityMultiplier)
			totalQualityBonus += scaledModuleBonus(moduleID, effects.QualityBonus, moduleQualityMultiplier)
			totalQualityBonus += effects.QualityPenalty
		}

		speedMultiplier := math.Max(0.2, 1+(totalSpeedPercentModifier/100))
		effectiveCraftSpeed := (baseCraftSpeed * machineQualityMultiplier) * speedMultiplier
		effectiveProductivity := math.Max(0, baseProductivity+totalProductivityBonus)
		effectiveQuality := math.Max(0, baseQuality+totalQualityBonus)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		var b strings.Builder
		b.WriteString(renderItemStatus("Updated machine stats", "is-primary", false))
		b.WriteString(renderMachineStatFields(
			effectiveProductivity,
			effectiveCraftSpeed,
			effectiveQuality,
			baseProductivity,
			baseCraftSpeed,
			baseQuality,
			true,
		))
		_, _ = w.Write([]byte(b.String()))
	})
	mux.HandleFunc("/partials/recycler-stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		baseRecycleTimeSource := parseFloatDefault(
			r.URL.Query().Get("base_recycle_time_seconds_source"),
			recycleTimeSourceFromCraftTime(defaultRecipeCraftTimeSeconds),
		)
		recyclerStats := calculateRecyclerStats(r.URL.Query(), baseRecycleTimeSource)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		var b strings.Builder
		b.WriteString(renderItemStatus("Updated recycler stats", "is-primary", false))
		b.WriteString(renderRecyclerStatFields(
			recyclerStats.EffectiveCraftSpeed,
			recyclerStats.EffectiveQuality,
			recyclerStats.EffectiveRecycleTimeSeconds,
			recyclerStats.BaseCraftSpeed,
			recyclerStats.BaseQuality,
			recyclerStats.BaseRecycleTimeSeconds,
			true,
		))
		_, _ = w.Write([]byte(b.String()))
	})
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/plan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		targetQuality, err := parseQualityTier(r.FormValue("target_quality"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`<p class="nes-text is-error">Invalid target quality.</p>`))
			return
		}

		changedQualityValue := strings.TrimSpace(r.FormValue("changed_quality"))
		if changedQualityValue == "" {
			changedQualityValue = "normal"
		}

		machineProductivity, err := parseFloat(r.FormValue("machine_productivity"), "machine productivity")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`<p class="nes-text is-error">Invalid machine productivity.</p>`))
			return
		}
		machineProductivity = clampFloat(machineProductivity, minMachineProductivityPct, maxMachineProductivityPct)

		machineCraftSpeed, err := parseFloat(r.FormValue("machine_craft_speed"), "machine craft speed")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`<p class="nes-text is-error">Invalid machine craft speed.</p>`))
			return
		}
		machineCraftSpeed = clampFloat(machineCraftSpeed, minMachineCraftSpeed, maxMachineCraftSpeed)

		machineQualityPercentage, err := parseFloat(r.FormValue("machine_quality_percentage"), "machine quality percentage")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`<p class="nes-text is-error">Invalid machine quality percentage.</p>`))
			return
		}
		machineQualityPercentage = clampFloat(machineQualityPercentage, minMachineQualityPct, maxMachineQualityPct)

		baseOutputPerCraft, err := parseFloat(r.FormValue("base_output_per_craft"), "base output per craft")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`<p class="nes-text is-error">Invalid base output per craft.</p>`))
			return
		}
		baseOutputPerCraft = clampFloat(baseOutputPerCraft, minBaseOutputPerCraft, maxBaseOutputPerCraft)

		baseCraftTimeSeconds, err := parseFloat(r.FormValue("base_craft_time_seconds"), "base craft time seconds")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`<p class="nes-text is-error">Invalid base craft time.</p>`))
			return
		}
		baseCraftTimeSeconds = clampFloat(baseCraftTimeSeconds, minBaseCraftTimeSec, maxBaseCraftTimeSec)

		recyclerCraftSpeed, err := parseFloat(r.FormValue("recycler_craft_speed"), "recycler craft speed")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`<p class="nes-text is-error">Invalid recycler craft speed.</p>`))
			return
		}
		recyclerCraftSpeed = clampFloat(recyclerCraftSpeed, minRecyclerCraftSpeed, maxRecyclerCraftSpeed)

		recyclerQualityPercentage, err := parseFloat(r.FormValue("recycler_quality_percentage"), "recycler quality percentage")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`<p class="nes-text is-error">Invalid recycler quality percentage.</p>`))
			return
		}
		recyclerQualityPercentage = clampFloat(recyclerQualityPercentage, minRecyclerQualityPct, maxRecyclerQualityPct)

		baseRecycleInputPerCycle, err := parseFloat(r.FormValue("base_recycle_input_per_cycle"), "base recycle input per cycle")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`<p class="nes-text is-error">Invalid recycle input per cycle.</p>`))
			return
		}
		baseRecycleInputPerCycle = clampFloat(baseRecycleInputPerCycle, minRecycleInputPerCycle, maxRecycleInputPerCycle)

		baseRecycleTimeSeconds, err := parseFloat(r.FormValue("base_recycle_time_seconds"), "base recycle time seconds")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`<p class="nes-text is-error">Invalid base recycle time.</p>`))
			return
		}
		baseRecycleTimeSeconds = clampFloat(baseRecycleTimeSeconds, minRecycleTimeSec, maxRecycleTimeSec)

		planInput := internal.PlanInput{
			TargetQuality: targetQuality,
			Machine: internal.Machine{
				Productivity:      machineProductivity,
				CraftSpeed:        machineCraftSpeed,
				QualityPercentage: machineQualityPercentage,
			},
			Recycler: internal.Recycler{
				CraftSpeed:        recyclerCraftSpeed,
				QualityPercentage: recyclerQualityPercentage,
			},
			BaseOutputPerCraft:       baseOutputPerCraft,
			BaseCraftTimeSeconds:     baseCraftTimeSeconds,
			BaseRecycleInputPerCycle: baseRecycleInputPerCycle,
			BaseRecycleTimeSeconds:   baseRecycleTimeSeconds,
		}

		var plan internal.PlanResult
		if strings.EqualFold(changedQualityValue, "total") {
			totalMachines, err := parseInt(r.FormValue("total_machines"), "total_machines")
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`<p class="nes-text is-error">Invalid total machines.</p>`))
				return
			}
			totalMachines = clampInt(totalMachines, minMachineCount, maxMachineCount)

			plan, err = internal.BuildPlanFromTotalMachines(planInput, totalMachines)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = fmt.Fprintf(
					w,
					`<p class="nes-text is-error">Unable to build plan: %s</p>`,
					html.EscapeString(err.Error()),
				)
				return
			}
		} else {
			changedQuality, err := parseQualityTier(changedQualityValue)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`<p class="nes-text is-error">Invalid changed quality field.</p>`))
				return
			}

			changedMachineField := fmt.Sprintf("%s_machines", strings.ToLower(changedQuality.String()))
			anchorMachineCount, err := parseInt(r.FormValue(changedMachineField), changedMachineField)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = fmt.Fprintf(
					w,
					`<p class="nes-text is-error">Invalid machine count for %s.</p>`,
					html.EscapeString(changedQuality.String()),
				)
				return
			}
			anchorMachineCount = clampInt(anchorMachineCount, minMachineCount, maxMachineCount)

			plan, err = internal.BuildPlanFromAnchorQuality(planInput, changedQuality, anchorMachineCount)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = fmt.Fprintf(
					w,
					`<p class="nes-text is-error">Unable to build plan: %s</p>`,
					html.EscapeString(err.Error()),
				)
				return
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(renderAllocationSection(plan)))
	})

	addr := ":8080"
	rateLimitConfig := loadRateLimitConfig()
	handler := withRequestLogging(
		withSecurityHeaders(withRateLimit(mux, rateLimitConfig)),
		rateLimitConfig.TrustProxy,
		log.Default(),
	)

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  parseEnvDuration("SERVER_READ_TIMEOUT", 10*time.Second),
		WriteTimeout: parseEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:  parseEnvDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
	}
	shutdownTimeout := parseEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 15*time.Second)

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("listening on http://localhost%s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErr:
		if err != nil {
			log.Fatal(err)
		}
		return
	case <-signalCtx.Done():
		log.Printf("shutdown signal received, draining active requests")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		if closeErr := server.Close(); closeErr != nil {
			log.Printf("forced server close failed: %v", closeErr)
		}
	}

	if err := <-serverErr; err != nil {
		log.Fatal(err)
	}
}

func parseQualityTier(value string) (internal.QualityTier, error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "normal":
		return internal.QualityNormal, nil
	case "uncommon":
		return internal.QualityUncommon, nil
	case "rare":
		return internal.QualityRare, nil
	case "epic":
		return internal.QualityEpic, nil
	case "legendary":
		return internal.QualityLegendary, nil
	default:
		return internal.QualityTier(255), fmt.Errorf("invalid quality tier")
	}
}

func parseInt(value string, fieldName string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("invalid %s", fieldName)
	}
	return n, nil
}

func parseFloat(value string, fieldName string) (float64, error) {
	n, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", fieldName)
	}
	return n, nil
}

func parseFloatDefault(value string, fallback float64) float64 {
	n, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return fallback
	}
	return n
}

func renderAllocationSection(plan internal.PlanResult) string {
	tiers := []internal.QualityTier{
		internal.QualityNormal,
		internal.QualityUncommon,
		internal.QualityRare,
		internal.QualityEpic,
		internal.QualityLegendary,
	}

	var b strings.Builder
	b.WriteString(`<section class="machine-boxes"><h3>Machines Dedicated Per Quality</h3>`)
	_, _ = fmt.Fprintf(
		&b,
		`<div class="machine-box total-machine-box"><label for="total-machines">Total Machines</label><input id="total-machines" name="total_machines" class="nes-input" type="number" min="0" step="1" value="%d" hx-post="/plan" hx-trigger="change, keyup delay:300ms" hx-target="#allocation-result" hx-swap="innerHTML" hx-include="#planner-form, #allocation-result input" hx-vals='{"changed_quality":"total"}'></div>`,
		plan.TotalMachines,
	)
	for _, tier := range tiers {
		slug := strings.ToLower(tier.String())
		ratio := plan.MachineRatioByQuality[tier] * 100
		_, _ = fmt.Fprintf(
			&b,
			`<div class="machine-box"><label for="%s-machines">%s</label><input id="%s-machines" name="%s_machines" class="nes-input" type="number" min="0" step="1" value="%d" hx-post="/plan" hx-trigger="change, keyup delay:300ms" hx-target="#allocation-result" hx-swap="innerHTML" hx-include="#planner-form" hx-vals='{"changed_quality":"%s"}'><p class="machine-ratio nes-text is-primary">Allocation Ratio: %.2f%%</p></div>`,
			slug,
			tier.String(),
			slug,
			slug,
			plan.MachinesByQuality[tier],
			slug,
			ratio,
		)
	}
	targetPerSecond := math.Max(0, plan.ProducedPerSecond-plan.RecycleLoadPerSecond)
	targetPerHour := targetPerSecond * 3600
	recycledPerHour := plan.RecycleLoadPerSecond * 3600
	_, _ = fmt.Fprintf(
		&b,
		`<p class="nes-text is-success">Recyclers Needed: %d</p><p class="nes-text is-primary">Total Crafted Output (items/s): %.4f</p><p class="nes-text is-primary">Items Sent to Recyclers (items/s): %.4f</p><p class="nes-text is-primary">Recycler Capacity per Machine (items/s): %.4f</p><p class="nes-text is-primary">Target Quality Output (items/hour): %.2f</p><p class="nes-text is-primary">Total Items Recycled (items/hour): %.2f</p></section>`,
		plan.RequiredRecyclers,
		plan.ProducedPerSecond,
		plan.RecycleLoadPerSecond,
		plan.RecyclerPerSecond,
		targetPerHour,
		recycledPerHour,
	)
	b.WriteString(renderCalculationWork(plan))
	return b.String()
}

func renderCalculationWork(plan internal.PlanResult) string {
	tiers := []internal.QualityTier{
		internal.QualityNormal,
		internal.QualityUncommon,
		internal.QualityRare,
		internal.QualityEpic,
		internal.QualityLegendary,
	}

	targetPerSecond := math.Max(0, plan.ProducedPerSecond-plan.RecycleLoadPerSecond)
	recyclerQuotient := 0.0
	if plan.RecyclerPerSecond > 0 {
		recyclerQuotient = plan.RecycleLoadPerSecond / plan.RecyclerPerSecond
	}

	var b strings.Builder
	b.WriteString(`<details class="calc-work"><summary>Show Your Work</summary><div class="calc-work-content">`)
	_, _ = fmt.Fprintf(
		&b,
		`<p class="calc-line">Target Quality Output (items/s) = Total Crafted Output - Items Sent to Recyclers = %.4f - %.4f = %.4f</p>`,
		plan.ProducedPerSecond,
		plan.RecycleLoadPerSecond,
		targetPerSecond,
	)
	_, _ = fmt.Fprintf(
		&b,
		`<p class="calc-line">Recyclers Needed = ceil(Items Sent to Recyclers / Recycler Capacity per Machine) = ceil(%.4f / %.4f) = ceil(%.4f) = %d</p>`,
		plan.RecycleLoadPerSecond,
		plan.RecyclerPerSecond,
		recyclerQuotient,
		plan.RequiredRecyclers,
	)
	_, _ = fmt.Fprintf(
		&b,
		`<p class="calc-line">Machine allocation uses: Machines for tier = ceil(Total Machines Exact * Tier Ratio). Total Machines Exact = %.4f</p>`,
		plan.TotalMachinesExact,
	)
	for _, tier := range tiers {
		ratio := plan.MachineRatioByQuality[tier]
		rawMachines := plan.TotalMachinesExact * ratio
		_, _ = fmt.Fprintf(
			&b,
			`<p class="calc-line">%s Machines = ceil(%.4f * %.6f) = ceil(%.4f) = %d</p>`,
			html.EscapeString(tier.String()),
			plan.TotalMachinesExact,
			ratio,
			rawMachines,
			plan.MachinesByQuality[tier],
		)
	}
	b.WriteString(`</div></details>`)
	return b.String()
}

func loadRecyclableItems(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	items := []string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		item := strings.TrimSpace(scanner.Text())
		if item == "" {
			continue
		}
		items = append(items, item)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func parseMaxUnlockedQuality(value string) internal.QualityTier {
	tier, err := parseQualityTier(value)
	if err != nil {
		return internal.QualityLegendary
	}
	return tier
}

func selectMachine(producers []string, preferred string) string {
	preferred = strings.TrimSpace(preferred)
	if preferred == "" {
		if len(producers) == 0 {
			return ""
		}
		return producers[0]
	}

	for _, producer := range producers {
		if strings.EqualFold(strings.TrimSpace(producer), preferred) {
			return producer
		}
	}

	if len(producers) == 0 {
		return ""
	}
	return producers[0]
}

func resolveMachineDetails(machine string, cache *machineDetailsCache) (MachineWikiDetails, error) {
	if strings.TrimSpace(machine) == "" {
		return MachineWikiDetails{
			Machine:                "",
			CraftSpeed:             1,
			ProductivityPercentage: 0,
			QualityPercentage:      0,
			ModuleSlots:            0,
		}, nil
	}

	details, err := cache.GetOrFetch(machine, fetchMachineDetailsFromWiki)
	if err != nil {
		return MachineWikiDetails{
			Machine:                machine,
			CraftSpeed:             1,
			ProductivityPercentage: 0,
			QualityPercentage:      0,
			ModuleSlots:            0,
		}, err
	}

	if details.ModuleSlots < 0 {
		details.ModuleSlots = 0
	}

	return details, nil
}

func renderItemStatus(message string, className string, oob bool) string {
	oobAttr := ""
	if oob {
		oobAttr = ` hx-swap-oob="outerHTML"`
	}
	return fmt.Sprintf(
		`<p id="item-load-status" class="nes-text %s"%s>%s</p>`,
		html.EscapeString(className),
		oobAttr,
		html.EscapeString(message),
	)
}

func renderRecipeFields(baseOutputPerCraft float64, baseCraftTimeSeconds float64, oob bool) string {
	oobAttr := ""
	if oob {
		oobAttr = ` hx-swap-oob="outerHTML"`
	}

	outputValue := formatStatValue(baseOutputPerCraft)
	craftTimeValue := formatStatValue(baseCraftTimeSeconds)

	return fmt.Sprintf(
		`<div id="recipe-output-field" class="nes-field"%s><label for="base-output-per-craft">Recipe Output Per Craft</label><input id="base-output-per-craft" name="base_output_per_craft" class="nes-input" type="text" value="%s" data-base-value="%s" required readonly></div><div id="recipe-time-field" class="nes-field"%s><label for="base-craft-time-seconds">Recipe Craft Time (s)</label><input id="base-craft-time-seconds" name="base_craft_time_seconds" class="nes-input" type="text" value="%s" data-base-value="%s" required readonly></div>`,
		oobAttr,
		html.EscapeString(outputValue),
		html.EscapeString(outputValue),
		oobAttr,
		html.EscapeString(craftTimeValue),
		html.EscapeString(craftTimeValue),
	)
}

func renderCraftMachineField(producers []string, selectedMachine string, oob bool) string {
	oobAttr := ""
	if oob {
		oobAttr = ` hx-swap-oob="outerHTML"`
	}

	var options strings.Builder
	if len(producers) == 0 {
		options.WriteString(`<option value="">No Crafting Machines Found</option>`)
	} else {
		for _, producer := range producers {
			selectedAttr := ""
			if strings.EqualFold(strings.TrimSpace(producer), strings.TrimSpace(selectedMachine)) {
				selectedAttr = ` selected`
			}
			_, _ = fmt.Fprintf(
				&options,
				`<option value="%s"%s>%s</option>`,
				html.EscapeString(producer),
				selectedAttr,
				html.EscapeString(producer),
			)
		}
	}

	return fmt.Sprintf(
		`<div id="craft-machine-field" class="nes-field full-row"%s><label for="craft-machine">Crafting Machine</label><div class="nes-select"><select id="craft-machine" name="craft_machine" hx-get="/partials/machine-details" hx-trigger="change" hx-target="#item-load-status" hx-swap="outerHTML" hx-include="#planner-form">%s</select></div></div>`,
		oobAttr,
		options.String(),
	)
}

func renderMachineModuleField(moduleSlots int, hasMachine bool, maxUnlocked internal.QualityTier, oob bool) string {
	oobAttr := ""
	if oob {
		oobAttr = ` hx-swap-oob="outerHTML"`
	}

	var b strings.Builder
	_, _ = fmt.Fprintf(
		&b,
		`<div id="machine-module-field" class="nes-field full-row"%s><label>Machine Modules By Slot</label><div id="machine-module-slots" class="module-slot-grid">`,
		oobAttr,
	)

	if moduleSlots <= 0 {
		message := "Select a crafting machine to load module slots."
		if hasMachine {
			message = "This machine has no module slots."
		}
		_, _ = fmt.Fprintf(&b, `<p class="nes-text is-primary">%s</p>`, html.EscapeString(message))
		b.WriteString(`</div></div>`)
		return b.String()
	}

	moduleOptions := []struct {
		Value string
		Label string
	}{
		{Value: "", Label: "No Module"},
		{Value: "speed-module", Label: "Speed Module"},
		{Value: "speed-module-2", Label: "Speed Module 2"},
		{Value: "speed-module-3", Label: "Speed Module 3"},
		{Value: "productivity-module", Label: "Productivity Module"},
		{Value: "productivity-module-2", Label: "Productivity Module 2"},
		{Value: "productivity-module-3", Label: "Productivity Module 3"},
		{Value: "quality-module", Label: "Quality Module"},
		{Value: "quality-module-2", Label: "Quality Module 2"},
		{Value: "quality-module-3", Label: "Quality Module 3"},
	}

	qualityOptions := []struct {
		Tier  internal.QualityTier
		Value string
		Label string
	}{
		{Tier: internal.QualityNormal, Value: "normal", Label: "Normal"},
		{Tier: internal.QualityUncommon, Value: "uncommon", Label: "Uncommon"},
		{Tier: internal.QualityRare, Value: "rare", Label: "Rare"},
		{Tier: internal.QualityEpic, Value: "epic", Label: "Epic"},
		{Tier: internal.QualityLegendary, Value: "legendary", Label: "Legendary"},
	}

	for i := 1; i <= moduleSlots; i++ {
		_, _ = fmt.Fprintf(
			&b,
			`<div class="module-slot-row"><div class="nes-field"><label for="machine-module-slot-%d">Module Slot %d</label><div class="nes-select"><select id="machine-module-slot-%d" name="machine_module_slot_%d" hx-get="/partials/module-stats" hx-trigger="change" hx-target="#item-load-status" hx-swap="outerHTML" hx-include="#planner-form">`,
			i,
			i,
			i,
			i,
		)
		for _, moduleOption := range moduleOptions {
			_, _ = fmt.Fprintf(
				&b,
				`<option value="%s">%s</option>`,
				html.EscapeString(moduleOption.Value),
				html.EscapeString(moduleOption.Label),
			)
		}
		_, _ = fmt.Fprintf(
			&b,
			`</select></div></div><div class="nes-field"><label for="machine-module-slot-%d-quality">Slot %d Module Quality</label><div class="nes-select"><select id="machine-module-slot-%d-quality" name="machine_module_slot_%d_quality" data-slot-index="%d" hx-get="/partials/module-stats" hx-trigger="change" hx-target="#item-load-status" hx-swap="outerHTML" hx-include="#planner-form" disabled>`,
			i,
			i,
			i,
			i,
			i,
		)
		for _, qualityOption := range qualityOptions {
			if qualityOption.Tier > maxUnlocked {
				continue
			}
			_, _ = fmt.Fprintf(
				&b,
				`<option value="%s">%s</option>`,
				html.EscapeString(qualityOption.Value),
				html.EscapeString(qualityOption.Label),
			)
		}
		b.WriteString(`</select></div></div></div>`)
	}

	b.WriteString(`</div></div>`)
	return b.String()
}

func renderMachineStatFields(
	productivity float64,
	craftSpeed float64,
	quality float64,
	baseProductivity float64,
	baseCraftSpeed float64,
	baseQuality float64,
	oob bool,
) string {
	oobAttr := ""
	if oob {
		oobAttr = ` hx-swap-oob="outerHTML"`
	}

	productivityValue := formatStatValue(productivity)
	craftSpeedValue := formatStatValue(craftSpeed)
	qualityValue := formatStatValue(quality)
	baseProductivityValue := formatStatValue(baseProductivity)
	baseCraftSpeedValue := formatStatValue(baseCraftSpeed)
	baseQualityValue := formatStatValue(baseQuality)

	return fmt.Sprintf(
		`<div id="machine-productivity-field" class="nes-field"%s><label for="machine-productivity">Machine Productivity (%%)</label><input id="machine-productivity" name="machine_productivity" class="nes-input" type="text" value="%s" data-base-value="%s" required readonly></div><div id="machine-craft-speed-field" class="nes-field"%s><label for="machine-craft-speed">Machine Craft Speed</label><input id="machine-craft-speed" name="machine_craft_speed" class="nes-input" type="text" value="%s" data-base-value="%s" required readonly></div><div id="machine-quality-percentage-field" class="nes-field"%s><label for="machine-quality-percentage">Machine Quality (%%)</label><input id="machine-quality-percentage" name="machine_quality_percentage" class="nes-input" type="text" value="%s" data-base-value="%s" required readonly></div>`,
		oobAttr,
		html.EscapeString(productivityValue),
		html.EscapeString(baseProductivityValue),
		oobAttr,
		html.EscapeString(craftSpeedValue),
		html.EscapeString(baseCraftSpeedValue),
		oobAttr,
		html.EscapeString(qualityValue),
		html.EscapeString(baseQualityValue),
	)
}

func renderRecyclerStatFields(
	craftSpeed float64,
	quality float64,
	recycleTimeSeconds float64,
	baseCraftSpeed float64,
	baseQuality float64,
	baseRecycleTimeSeconds float64,
	oob bool,
) string {
	oobAttr := ""
	if oob {
		oobAttr = ` hx-swap-oob="outerHTML"`
	}

	craftSpeedValue := formatStatValue(craftSpeed)
	qualityValue := formatStatValue(quality)
	recycleTimeValue := formatStatValue(recycleTimeSeconds)
	baseCraftSpeedValue := formatStatValue(baseCraftSpeed)
	baseQualityValue := formatStatValue(baseQuality)
	baseRecycleTimeValue := formatStatValue(baseRecycleTimeSeconds)

	return fmt.Sprintf(
		`<div id="recycler-craft-speed-field" class="nes-field"%s><label for="recycler-craft-speed">Recycler Craft Speed</label><input id="recycler-craft-speed" name="recycler_craft_speed" class="nes-input" type="text" value="%s" data-base-value="%s" required readonly></div><div id="recycler-quality-percentage-field" class="nes-field"%s><label for="recycler-quality-percentage">Recycler Quality (%%)</label><input id="recycler-quality-percentage" name="recycler_quality_percentage" class="nes-input" type="text" value="%s" data-base-value="%s" required readonly></div><div id="recycle-time-field" class="nes-field"%s><label for="base-recycle-time-seconds">Recycle Cycle Time (s)</label><input id="base-recycle-time-seconds" name="base_recycle_time_seconds" class="nes-input" type="text" value="%s" data-base-value="%s" required readonly></div>`,
		oobAttr,
		html.EscapeString(craftSpeedValue),
		html.EscapeString(baseCraftSpeedValue),
		oobAttr,
		html.EscapeString(qualityValue),
		html.EscapeString(baseQualityValue),
		oobAttr,
		html.EscapeString(recycleTimeValue),
		html.EscapeString(baseRecycleTimeValue),
	)
}

func renderRecycleTimeSourceField(recycleTimeSource float64, oob bool) string {
	oobAttr := ""
	if oob {
		oobAttr = ` hx-swap-oob="outerHTML"`
	}

	value := formatStatValue(recycleTimeSource)
	return fmt.Sprintf(
		`<input id="base-recycle-time-seconds-source" name="base_recycle_time_seconds_source" type="hidden" value="%s"%s>`,
		html.EscapeString(value),
		oobAttr,
	)
}

type recyclerStats struct {
	EffectiveCraftSpeed         float64
	EffectiveQuality            float64
	EffectiveRecycleTimeSeconds float64
	BaseCraftSpeed              float64
	BaseQuality                 float64
	BaseRecycleTimeSeconds      float64
}

func recycleTimeSourceFromCraftTime(baseCraftTimeSeconds float64) float64 {
	if baseCraftTimeSeconds <= 0 {
		baseCraftTimeSeconds = defaultRecipeCraftTimeSeconds
	}
	return baseCraftTimeSeconds / recycleTimeDivisor
}

func calculateRecyclerStats(values url.Values, baseRecycleTimeSeconds float64) recyclerStats {
	baseCraftSpeed := clampFloat(parseFloatDefault(values.Get("base_recycler_craft_speed"), 0.5), minRecyclerCraftSpeed, maxRecyclerCraftSpeed)
	baseQuality := clampFloat(parseFloatDefault(values.Get("base_recycler_quality_percentage"), 0), minRecyclerQualityPct, maxRecyclerQualityPct)
	baseRecycleTimeSeconds = clampFloat(baseRecycleTimeSeconds, minRecycleTimeSec, maxRecycleTimeSec)
	if baseRecycleTimeSeconds <= 0 {
		baseRecycleTimeSeconds = recycleTimeSourceFromCraftTime(defaultRecipeCraftTimeSeconds)
	}

	recyclerQuality := strings.ToLower(strings.TrimSpace(values.Get("recycler_quality")))
	recyclerQualityMultiplier := qualityMultiplierByTierKey[recyclerQuality]
	if recyclerQualityMultiplier <= 0 {
		recyclerQualityMultiplier = 1
	}

	totalSpeedPercentModifier := 0.0
	totalQualityBonus := 0.0
	for i := 1; i <= 4; i++ {
		moduleID := strings.TrimSpace(values.Get(fmt.Sprintf("recycler_module_slot_%d", i)))
		if moduleID == "" {
			continue
		}

		effects, ok := moduleEffectsByID[moduleID]
		if !ok {
			continue
		}

		moduleQuality := strings.ToLower(strings.TrimSpace(values.Get(fmt.Sprintf("recycler_module_slot_%d_quality", i))))
		moduleQualityMultiplier := qualityMultiplierByTierKey[moduleQuality]
		if moduleQualityMultiplier <= 0 {
			moduleQualityMultiplier = 1
		}

		totalSpeedPercentModifier += scaledModuleBonus(moduleID, effects.SpeedBonus, moduleQualityMultiplier)
		totalSpeedPercentModifier += effects.SpeedPenalty
		totalQualityBonus += scaledModuleBonus(moduleID, effects.QualityBonus, moduleQualityMultiplier)
		totalQualityBonus += effects.QualityPenalty
	}

	speedMultiplier := math.Max(0.2, 1+(totalSpeedPercentModifier/100))
	effectiveCraftSpeed := (baseCraftSpeed * recyclerQualityMultiplier) * speedMultiplier
	effectiveQuality := math.Max(0, baseQuality+totalQualityBonus)

	effectiveRecycleTimeSeconds := baseRecycleTimeSeconds
	if effectiveCraftSpeed > 0 {
		effectiveRecycleTimeSeconds = baseRecycleTimeSeconds / effectiveCraftSpeed
	}

	return recyclerStats{
		EffectiveCraftSpeed:         effectiveCraftSpeed,
		EffectiveQuality:            effectiveQuality,
		EffectiveRecycleTimeSeconds: effectiveRecycleTimeSeconds,
		BaseCraftSpeed:              baseCraftSpeed,
		BaseQuality:                 baseQuality,
		BaseRecycleTimeSeconds:      baseRecycleTimeSeconds,
	}
}

func formatStatValue(value float64) string {
	rounded := math.Round(value*1000) / 1000
	return strconv.FormatFloat(rounded, 'f', -1, 64)
}
