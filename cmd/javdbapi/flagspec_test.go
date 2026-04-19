package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	javdbapi "github.com/RPbro/javdbapi"
)

func TestParseHomeFilterPreservesOmittedSemantics(t *testing.T) {
	t.Parallel()

	filter, err := parseHomeFilter("")
	require.NoError(t, err)
	assert.Equal(t, javdbapi.HomeFilter(""), filter)

	filter, err = parseHomeFilter("all")
	require.NoError(t, err)
	assert.Equal(t, javdbapi.HomeFilter(""), filter)

	_, err = parseHomeFilter("1")
	require.Error(t, err)
	assert.Equal(t, `invalid --filter "1": must be one of all, download, cnsub, review`, err.Error())
}

func TestParseActorFiltersSupportsCanonicalAndLegacyAliases(t *testing.T) {
	t.Parallel()

	filters, err := parseActorFilters("cnsub,download")
	require.NoError(t, err)
	assert.Equal(t, []javdbapi.ActorFilter{"c", "d"}, filters)

	filters, err = parseActorFilters("c,d")
	require.NoError(t, err)
	assert.Equal(t, []javdbapi.ActorFilter{"c", "d"}, filters)

	filters, err = parseActorFilters("download,download,cnsub")
	require.NoError(t, err)
	assert.Equal(t, []javdbapi.ActorFilter{"d", "c"}, filters)

	_, err = parseActorFilters("all,download")
	require.Error(t, err)
	assert.Equal(t, `invalid --filter "all,download": all cannot be combined with other values`, err.Error())
}

func TestParseActorFiltersRejectsCaseVariants(t *testing.T) {
	t.Parallel()

	_, err := parseActorFilters("Download")
	require.Error(t, err)
	assert.Equal(t, `invalid --filter "Download": must be one of all, playable, single, download, cnsub`, err.Error())
}

func TestSharedOutputSpecRejectsUnknownValue(t *testing.T) {
	t.Parallel()

	_, err := parseOutputMode("invalid")
	require.Error(t, err)
	assert.Equal(t, `invalid --output "invalid": must be one of file, console, both`, err.Error())
}

func TestAdditionalEnumSpecsCoverRemainingCanonicalValues(t *testing.T) {
	t.Parallel()

	homeType, err := parseHomeType("all")
	require.NoError(t, err)
	assert.Equal(t, javdbapi.HomeType(""), homeType)

	homeType, err = parseHomeType("western")
	require.NoError(t, err)
	assert.Equal(t, javdbapi.HomeType("western"), homeType)

	_, err = parseHomeType("Censored")
	require.Error(t, err)
	assert.Equal(t, `invalid --type "Censored": must be one of all, censored, uncensored, western`, err.Error())

	homeSort, err := parseHomeSort("publish")
	require.NoError(t, err)
	assert.Equal(t, javdbapi.HomeSort(""), homeSort)

	_, err = parseHomeSort("2")
	require.Error(t, err)
	assert.Equal(t, `invalid --sort "2": must be one of publish, magnet`, err.Error())

	makerFilter, err := parseMakerFilter("preview")
	require.NoError(t, err)
	assert.Equal(t, javdbapi.MakerFilter("preview"), makerFilter)

	makerFilter, err = parseMakerFilter("all")
	require.NoError(t, err)
	assert.Equal(t, javdbapi.MakerFilter(""), makerFilter)

	_, err = parseMakerFilter("clip")
	require.Error(t, err)
	assert.Equal(t, `invalid --filter "clip": must be one of all, playable, single, download, cnsub, preview`, err.Error())

	rankingPeriod, err := parseRankingPeriod("weekly")
	require.NoError(t, err)
	assert.Equal(t, javdbapi.RankingPeriod("weekly"), rankingPeriod)

	_, err = parseRankingPeriod("yearly")
	require.Error(t, err)
	assert.Equal(t, `invalid --period "yearly": must be one of daily, weekly, monthly`, err.Error())

	rankingType, err := parseRankingType("censored")
	require.NoError(t, err)
	assert.Equal(t, javdbapi.RankingType("censored"), rankingType)

	_, err = parseRankingType("asia")
	require.Error(t, err)
	assert.Equal(t, `invalid --type "asia": must be one of censored, uncensored, western`, err.Error())
}
