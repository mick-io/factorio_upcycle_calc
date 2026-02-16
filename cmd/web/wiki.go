package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const factorioWikiAPIBase = "https://wiki.factorio.com/api.php"

var (
	recipeLineRe            = regexp.MustCompile(`(?m)^\|recipe\s*=\s*(.+)$`)
	recipeTimeRe            = regexp.MustCompile(`(?i)\bTime,\s*([0-9]+(?:\.[0-9]+)?)`)
	recipeOutRe             = regexp.MustCompile(`=\s*[^,=\n]+,\s*([0-9]+(?:\.[0-9]+)?)`)
	producersLineRe         = regexp.MustCompile(`(?m)^\|producers\s*=\s*(.+)$`)
	spaceAgeProducersLineRe = regexp.MustCompile(`(?m)^\|space-age-producers\s*=\s*(.+)$`)
	firstNumberRe           = regexp.MustCompile(`[-+]?[0-9]*\.?[0-9]+`)
	translationRe           = regexp.MustCompile(`\{\{Translation\|([^}|]+)(?:\|[^}]*)?\}\}`)
	linkRe                  = regexp.MustCompile(`\[\[([^|\]]+)(?:\|([^\]]+))?\]\]`)
	templateRe              = regexp.MustCompile(`\{\{[^|}]+\|([^}|]+)(?:\|[^}]*)?\}\}`)
)

var machineInfoboxFallbacks = map[string][]string{
	"assembling machine": {"Assembling_machine_3", "Assembling_machine_2", "Assembling_machine_1"},
	"furnace":            {"Electric_furnace", "Steel_furnace", "Stone_furnace"},
}

type ItemWikiDetails struct {
	Item                 string   `json:"item"`
	BaseCraftTimeSeconds float64  `json:"base_craft_time_seconds"`
	BaseOutputPerCraft   float64  `json:"base_output_per_craft"`
	Producers            []string `json:"producers"`
	ProducersParsed      bool     `json:"producers_parsed"`
	SourcePage           string   `json:"source_page"`
}

type MachineWikiDetails struct {
	Machine                string  `json:"machine"`
	CraftSpeed             float64 `json:"craft_speed"`
	ProductivityPercentage float64 `json:"productivity_percentage"`
	QualityPercentage      float64 `json:"quality_percentage"`
	ModuleSlots            int     `json:"module_slots"`
	SourcePage             string  `json:"source_page"`
	DataParsed             bool    `json:"data_parsed"`
	ModuleSlotsParsed      bool    `json:"module_slots_parsed"`
	CacheVersion           int     `json:"cache_version"`
}

type wikiParseResponse struct {
	Parse struct {
		Title    string `json:"title"`
		Wikitext string `json:"wikitext"`
	} `json:"parse"`
	Error *struct {
		Info string `json:"info"`
	} `json:"error,omitempty"`
}

func fetchItemDetailsFromWiki(item string) (ItemWikiDetails, error) {
	item = strings.TrimSpace(item)
	if item == "" {
		return ItemWikiDetails{}, fmt.Errorf("item is required")
	}

	baseTitle := strings.ReplaceAll(item, "-", "_")
	candidates := []string{
		"Infobox:" + baseTitle,
	}

	for _, title := range candidates {
		wikitext, err := fetchWikiWikitext(title)
		if err != nil {
			continue
		}

		craftTime, outputAmount, err := parseRecipeFromWikitext(wikitext)
		if err != nil {
			continue
		}

		return ItemWikiDetails{
			Item:                 item,
			BaseCraftTimeSeconds: craftTime,
			BaseOutputPerCraft:   outputAmount,
			Producers:            parseProducersFromWikitext(wikitext),
			ProducersParsed:      true,
			SourcePage:           title,
		}, nil
	}

	return ItemWikiDetails{}, fmt.Errorf("unable to parse recipe data from wiki for %q", item)
}

func fetchWikiWikitext(pageTitle string) (string, error) {
	start := time.Now()
	metricsCollector.wikiFetchTotal.Add(1)
	defer func() {
		metricsCollector.wikiFetchLatencyNs.Add(uint64(time.Since(start).Nanoseconds()))
	}()

	values := url.Values{}
	values.Set("action", "parse")
	values.Set("page", pageTitle)
	values.Set("prop", "wikitext")
	values.Set("format", "json")
	values.Set("formatversion", "2")

	req, err := http.NewRequest(http.MethodGet, factorioWikiAPIBase+"?"+values.Encode(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "factorio-upcycle-calc/0.1 (+github.com/mick-io/factorio_upcycle_calc)")

	client := http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		metricsCollector.wikiFetchErrors.Add(1)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		metricsCollector.wikiFetchErrors.Add(1)
		return "", fmt.Errorf("wiki request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		metricsCollector.wikiFetchErrors.Add(1)
		return "", err
	}

	var parsed wikiParseResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		metricsCollector.wikiFetchErrors.Add(1)
		return "", err
	}
	if parsed.Error != nil {
		metricsCollector.wikiFetchErrors.Add(1)
		return "", fmt.Errorf("wiki parse error: %s", parsed.Error.Info)
	}
	if strings.TrimSpace(parsed.Parse.Wikitext) == "" {
		metricsCollector.wikiFetchErrors.Add(1)
		return "", fmt.Errorf("wiki page %q has empty wikitext", pageTitle)
	}
	return parsed.Parse.Wikitext, nil
}

func parseRecipeFromWikitext(wikitext string) (float64, float64, error) {
	recipeLineMatches := recipeLineRe.FindStringSubmatch(wikitext)
	if len(recipeLineMatches) != 2 {
		return 0, 0, fmt.Errorf("recipe line not found in wikitext")
	}
	recipeLine := strings.TrimSpace(recipeLineMatches[1])

	timeMatches := recipeTimeRe.FindStringSubmatch(recipeLine)
	if len(timeMatches) != 2 {
		return 0, 0, fmt.Errorf("recipe time not found in wikitext")
	}
	craftTime, err := strconv.ParseFloat(timeMatches[1], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid craft time: %w", err)
	}

	// Most infobox recipe lines omit explicit output amount when it is 1.
	outputAmount := 1.0
	outputMatches := recipeOutRe.FindStringSubmatch(recipeLine)
	if len(outputMatches) == 2 {
		parsedOutput, err := strconv.ParseFloat(outputMatches[1], 64)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid output amount: %w", err)
		}
		outputAmount = parsedOutput
	}

	return craftTime, outputAmount, nil
}

func parseProducersFromWikitext(wikitext string) []string {
	lines := []string{}

	if producersMatch := producersLineRe.FindStringSubmatch(wikitext); len(producersMatch) == 2 {
		lines = append(lines, producersMatch[1])
	}
	if spaceAgeMatch := spaceAgeProducersLineRe.FindStringSubmatch(wikitext); len(spaceAgeMatch) == 2 {
		lines = append(lines, spaceAgeMatch[1])
	}
	if len(lines) == 0 {
		return []string{}
	}

	seen := map[string]struct{}{}
	producers := []string{}

	for _, line := range lines {
		for _, token := range strings.Split(line, "+") {
			normalized := normalizeProducerName(token)
			if normalized == "" {
				continue
			}
			key := strings.ToLower(normalized)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			producers = append(producers, normalized)
		}
	}

	return producers
}

func normalizeProducerName(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}

	value = translationRe.ReplaceAllString(value, "$1")
	value = linkRe.ReplaceAllStringFunc(value, func(match string) string {
		submatches := linkRe.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}
		if strings.TrimSpace(submatches[2]) != "" {
			return submatches[2]
		}
		return submatches[1]
	})
	value = templateRe.ReplaceAllString(value, "$1")
	value = strings.TrimSpace(strings.Trim(value, "{}[]"))
	value = strings.ReplaceAll(value, "_", " ")
	if strings.EqualFold(value, "player") {
		return ""
	}
	return value
}

func fetchMachineDetailsFromWiki(machine string) (MachineWikiDetails, error) {
	machine = strings.TrimSpace(machine)
	if machine == "" {
		return MachineWikiDetails{}, fmt.Errorf("machine is required")
	}

	candidates := machineInfoboxCandidates(machine)
	for _, title := range candidates {
		wikitext, err := fetchWikiWikitext(title)
		if err != nil {
			continue
		}

		craftSpeed, ok := parseInfoboxNumericValue(wikitext, []string{
			"crafting-speed",
			"crafting_speed",
			"crafting speed",
		})
		if !ok {
			continue
		}

		productivity, _ := parseInfoboxNumericValue(wikitext, []string{
			"base-productivity",
			"productivity-bonus",
			"productivity",
		})
		quality, _ := parseInfoboxNumericValue(wikitext, []string{
			"quality-bonus",
			"quality",
		})
		moduleSlotsValue, moduleSlotsParsed := parseInfoboxNumericValue(wikitext, []string{
			"modules",
			"module-slots",
			"module_slots",
			"module slots",
		})
		if !moduleSlotsParsed {
			// Some generic infobox pages omit machine slot counts; keep searching specific pages.
			continue
		}
		moduleSlots := 0
		moduleSlots = int(moduleSlotsValue)
		if moduleSlots < 0 {
			moduleSlots = 0
		}

		return MachineWikiDetails{
			Machine:                machine,
			CraftSpeed:             craftSpeed,
			ProductivityPercentage: productivity,
			QualityPercentage:      quality,
			ModuleSlots:            moduleSlots,
			SourcePage:             title,
			DataParsed:             true,
			ModuleSlotsParsed:      moduleSlotsParsed,
			CacheVersion:           machineDetailsCacheVersion,
		}, nil
	}

	return MachineWikiDetails{}, fmt.Errorf("unable to parse machine data from wiki for %q", machine)
}

func machineInfoboxCandidates(machine string) []string {
	base := strings.TrimSpace(machine)
	if base == "" {
		return []string{}
	}

	candidates := []string{}
	seen := map[string]struct{}{}
	add := func(page string) {
		page = strings.TrimSpace(page)
		if page == "" {
			return
		}
		title := "Infobox:" + strings.ReplaceAll(strings.ReplaceAll(page, "-", "_"), " ", "_")
		if _, ok := seen[title]; ok {
			return
		}
		seen[title] = struct{}{}
		candidates = append(candidates, title)
	}

	add(base)
	if fallbackPages, ok := machineInfoboxFallbacks[strings.ToLower(base)]; ok {
		for _, page := range fallbackPages {
			add(page)
		}
	}

	return candidates
}

func parseInfoboxNumericValue(wikitext string, keys []string) (float64, bool) {
	for _, key := range keys {
		lineRe := regexp.MustCompile(`(?m)^\|` + regexp.QuoteMeta(key) + `\s*=\s*(.+)$`)
		matches := lineRe.FindStringSubmatch(wikitext)
		if len(matches) != 2 {
			continue
		}

		n, ok := parseFirstNumber(matches[1])
		if ok {
			return n, true
		}
	}
	return 0, false
}

func parseFirstNumber(value string) (float64, bool) {
	match := firstNumberRe.FindString(value)
	if match == "" {
		return 0, false
	}
	n, err := strconv.ParseFloat(match, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}
