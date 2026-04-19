package main

import (
	"fmt"
	"strings"

	cli "github.com/urfave/cli/v3"

	javdbapi "github.com/RPbro/javdbapi"
	"github.com/RPbro/javdbapi/internal/cliapp"
)

type enumSpec struct {
	description  string
	canonical    []string
	displayValue string
	aliases      map[string]string
}

const enumAllValue = "all"

func (s enumSpec) usage() string {
	usage := fmt.Sprintf("%s (%s)", s.description, strings.Join(s.canonical, "|"))
	if s.displayValue != "" {
		usage += fmt.Sprintf(" (default: %s)", s.displayValue)
	}
	return usage
}

func (s enumSpec) normalize(flagName string, raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if alias, ok := s.aliases[value]; ok {
		value = alias
	}
	for _, candidate := range s.canonical {
		if value == candidate {
			return value, nil
		}
	}
	return "", fmt.Errorf("invalid --%s %q: must be one of %s", flagName, raw, strings.Join(s.canonical, ", "))
}

var (
	homeTypeSpec = enumSpec{
		description:  "home type",
		canonical:    []string{enumAllValue, "censored", "uncensored", "western"},
		displayValue: enumAllValue,
	}
	homeFilterSpec = enumSpec{
		description:  "home filter",
		canonical:    []string{enumAllValue, "download", "cnsub", "review"},
		displayValue: enumAllValue,
	}
	homeSortSpec = enumSpec{
		description:  "home sort",
		canonical:    []string{"publish", "magnet"},
		displayValue: "publish",
	}
	makerFilterSpec = enumSpec{
		description:  "maker filter",
		canonical:    []string{enumAllValue, "playable", "single", "download", "cnsub", "preview"},
		displayValue: enumAllValue,
	}
	actorFilterSpec = enumSpec{
		description:  "actor filter",
		canonical:    []string{enumAllValue, "playable", "single", "download", "cnsub"},
		displayValue: enumAllValue,
		aliases: map[string]string{
			"p": "playable",
			"s": "single",
			"d": "download",
			"c": "cnsub",
		},
	}
	rankingPeriodSpec = enumSpec{
		description: "ranking period",
		canonical:   []string{"daily", "weekly", "monthly"},
	}
	rankingTypeSpec = enumSpec{
		description: "ranking type",
		canonical:   []string{"censored", "uncensored", "western"},
	}
	outputModeSpec = enumSpec{
		description:  "output mode",
		canonical:    []string{"file", "console", "both"},
		displayValue: "file",
	}
)

func parseHomeType(raw string) (javdbapi.HomeType, error) {
	value, err := homeTypeSpec.normalize("type", raw)
	if err != nil || value == "" || value == enumAllValue {
		return javdbapi.HomeType(""), err
	}
	return javdbapi.HomeType(value), nil
}

func parseHomeFilter(raw string) (javdbapi.HomeFilter, error) {
	value, err := homeFilterSpec.normalize("filter", raw)
	if err != nil || value == "" || value == enumAllValue {
		return javdbapi.HomeFilter(""), err
	}
	switch value {
	case "download":
		return javdbapi.HomeFilter("1"), nil
	case "cnsub":
		return javdbapi.HomeFilter("2"), nil
	case "review":
		return javdbapi.HomeFilter("3"), nil
	default:
		return javdbapi.HomeFilter(""), nil
	}
}

func parseHomeSort(raw string) (javdbapi.HomeSort, error) {
	value, err := homeSortSpec.normalize("sort", raw)
	if err != nil || value == "" || value == "publish" {
		return javdbapi.HomeSort(""), err
	}
	return javdbapi.HomeSort("2"), nil
}

func parseMakerFilter(raw string) (javdbapi.MakerFilter, error) {
	value, err := makerFilterSpec.normalize("filter", raw)
	if err != nil || value == "" || value == enumAllValue {
		return javdbapi.MakerFilter(""), err
	}
	return javdbapi.MakerFilter(value), nil
}

func parseActorFilters(raw string) ([]javdbapi.ActorFilter, error) {
	parts := parseCommaValues(raw)
	if len(parts) == 0 {
		return nil, nil
	}

	normalized := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		value, err := actorFilterSpec.normalize("filter", part)
		if err != nil {
			return nil, fmt.Errorf("invalid --filter %q: must be one of %s", raw, strings.Join(actorFilterSpec.canonical, ", "))
		}
		if value == enumAllValue && len(parts) > 1 {
			return nil, fmt.Errorf("invalid --filter %q: all cannot be combined with other values", raw)
		}
		if value == enumAllValue {
			return nil, nil
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	result := make([]javdbapi.ActorFilter, 0, len(normalized))
	for _, value := range normalized {
		switch value {
		case "playable":
			result = append(result, javdbapi.ActorFilter("p"))
		case "single":
			result = append(result, javdbapi.ActorFilter("s"))
		case "download":
			result = append(result, javdbapi.ActorFilter("d"))
		case "cnsub":
			result = append(result, javdbapi.ActorFilter("c"))
		}
	}
	return result, nil
}

func parseRankingPeriod(raw string) (javdbapi.RankingPeriod, error) {
	value, err := rankingPeriodSpec.normalize("period", raw)
	return javdbapi.RankingPeriod(value), err
}

func parseRankingType(raw string) (javdbapi.RankingType, error) {
	value, err := rankingTypeSpec.normalize("type", raw)
	return javdbapi.RankingType(value), err
}

func parseOutputMode(raw string) (cliapp.OutputMode, error) {
	value, err := outputModeSpec.normalize("output", raw)
	if err != nil {
		return "", err
	}
	if value == "" {
		value = outputModeSpec.displayValue
	}
	return cliapp.OutputMode(value), nil
}

func newStringFlag(name string, spec enumSpec, required bool) *cli.StringFlag {
	return &cli.StringFlag{Name: name, Usage: spec.usage(), Required: required}
}
