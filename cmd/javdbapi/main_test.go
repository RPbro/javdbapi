package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cli "github.com/urfave/cli/v3"

	javdbapi "github.com/RPbro/javdbapi"
	"github.com/RPbro/javdbapi/internal/cliapp"
)

type fakeFetcher struct {
	homeCalls    []javdbapi.HomeQuery
	searchCalls  []javdbapi.SearchQuery
	makerCalls   []javdbapi.MakerVideosQuery
	actorCalls   []javdbapi.ActorVideosQuery
	rankingCalls []javdbapi.RankingQuery
	detailCalls  []javdbapi.VideoID
	reviewCalls  []javdbapi.VideoID

	page    javdbapi.Page[javdbapi.VideoSummary]
	pageErr error

	detail    *javdbapi.VideoDetail
	detailErr error
	reviews   []javdbapi.Review
	reviewErr error
}

func (f *fakeFetcher) Home(_ context.Context, q javdbapi.HomeQuery) (javdbapi.Page[javdbapi.VideoSummary], error) {
	f.homeCalls = append(f.homeCalls, q)
	return f.page, f.pageErr
}

func (f *fakeFetcher) Search(_ context.Context, q javdbapi.SearchQuery) (javdbapi.Page[javdbapi.VideoSummary], error) {
	f.searchCalls = append(f.searchCalls, q)
	return f.page, f.pageErr
}

func (f *fakeFetcher) MakerVideos(_ context.Context, q javdbapi.MakerVideosQuery) (javdbapi.Page[javdbapi.VideoSummary], error) {
	f.makerCalls = append(f.makerCalls, q)
	return f.page, f.pageErr
}

func (f *fakeFetcher) ActorVideos(_ context.Context, q javdbapi.ActorVideosQuery) (javdbapi.Page[javdbapi.VideoSummary], error) {
	f.actorCalls = append(f.actorCalls, q)
	return f.page, f.pageErr
}

func (f *fakeFetcher) Ranking(_ context.Context, q javdbapi.RankingQuery) (javdbapi.Page[javdbapi.VideoSummary], error) {
	f.rankingCalls = append(f.rankingCalls, q)
	return f.page, f.pageErr
}

func (f *fakeFetcher) Detail(_ context.Context, id javdbapi.VideoID) (*javdbapi.VideoDetail, error) {
	f.detailCalls = append(f.detailCalls, id)
	return f.detail, f.detailErr
}

func (f *fakeFetcher) Reviews(_ context.Context, id javdbapi.VideoID) ([]javdbapi.Review, error) {
	f.reviewCalls = append(f.reviewCalls, id)
	return f.reviews, f.reviewErr
}

func stubBuilder(fetcher cliapp.Fetcher) fetcherBuilder {
	return func(*cli.Command, *slog.Logger) (cliapp.Fetcher, error) {
		return fetcher, nil
	}
}

func TestSearchCommandBuildsExpectedRequest(t *testing.T) {
	t.Parallel()

	fetcher := &fakeFetcher{}
	cmd := newCommand(stubBuilder(fetcher), io.Discard, io.Discard)

	err := cmd.Run(context.Background(), []string{
		"javdbapi",
		"search",
		"--keyword", "VR",
		"--page", "2",
		"--max-pages", "3",
		"--output", "console",
		"--stale-after", "48h",
	})
	require.NoError(t, err)
	require.Len(t, fetcher.searchCalls, 1)
	assert.Equal(t, javdbapi.SearchQuery{Keyword: "VR", Page: 2}, fetcher.searchCalls[0])
}

func TestSummaryOnlyFlagAvailableOnListCommandsButNotVideo(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"search", "home", "maker", "actor", "ranking"} {
		t.Run(name, func(t *testing.T) {
			var stdout bytes.Buffer
			cmd := newCommand(stubBuilder(&fakeFetcher{}), &stdout, io.Discard)
			err := cmd.Run(context.Background(), []string{"javdbapi", name, "--help"})
			require.NoError(t, err)
			assert.Contains(t, stdout.String(), "--summary-only")
		})
	}

	var stdout bytes.Buffer
	cmd := newCommand(stubBuilder(&fakeFetcher{}), &stdout, io.Discard)
	err := cmd.Run(context.Background(), []string{"javdbapi", "video", "--help"})
	require.NoError(t, err)
	assert.NotContains(t, stdout.String(), "--summary-only")
}

func TestSummaryOnlyOutputsNDJSONWithoutDetailOrReviewCalls(t *testing.T) {
	t.Parallel()

	fetcher := &fakeFetcher{
		page: javdbapi.Page[javdbapi.VideoSummary]{
			Items:   []javdbapi.VideoSummary{{ID: "P9Jkq9", Code: "SNOS-177"}},
			Number:  1,
			HasNext: false,
		},
	}
	var stdout bytes.Buffer
	cmd := newCommand(stubBuilder(fetcher), &stdout, io.Discard)

	err := cmd.Run(context.Background(), []string{
		"javdbapi",
		"search",
		"--keyword", "VR",
		"--summary-only",
		"--output", "console",
	})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), `"id":"P9Jkq9"`)
	assert.Contains(t, stdout.String(), `"code":"SNOS-177"`)
	assert.Empty(t, fetcher.detailCalls, "summary-only must never call Detail")
	assert.Empty(t, fetcher.reviewCalls, "summary-only must never call Reviews")
}

func TestSummaryOnlyLogsSummariesOutputCount(t *testing.T) {
	t.Parallel()

	fetcher := &fakeFetcher{
		page: javdbapi.Page[javdbapi.VideoSummary]{
			Items:   []javdbapi.VideoSummary{{ID: "a"}, {ID: "b"}},
			Number:  1,
			HasNext: false,
		},
	}
	var stdout, stderr bytes.Buffer
	cmd := newCommand(stubBuilder(fetcher), &stdout, &stderr)

	err := cmd.Run(context.Background(), []string{
		"javdbapi",
		"search",
		"--keyword", "VR",
		"--summary-only",
		"--output", "console",
	})
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "summaries_output=2")
}

func TestSummaryOnlyRejectsNonConsoleOutput(t *testing.T) {
	t.Parallel()

	cmd := newCommand(stubBuilder(&fakeFetcher{}), io.Discard, io.Discard)
	err := cmd.Run(context.Background(), []string{
		"javdbapi",
		"search",
		"--keyword", "VR",
		"--summary-only",
		"--output", "file",
	})
	require.Error(t, err)
	assert.Equal(t, "--summary-only requires --output console", err.Error())
}

func TestVideoCommandRejectsPaginationFlags(t *testing.T) {
	t.Parallel()

	cmd := newCommand(stubBuilder(&fakeFetcher{}), io.Discard, io.Discard)
	err := cmd.Run(context.Background(), []string{
		"javdbapi",
		"video",
		"--id", "ZNdEbV",
		"--page", "2",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "flag provided but not defined")
}

func TestVideoCommandRequiresID(t *testing.T) {
	t.Parallel()

	cmd := newCommand(stubBuilder(&fakeFetcher{}), io.Discard, io.Discard)
	err := cmd.Run(context.Background(), []string{"javdbapi", "video"})
	require.Error(t, err)
}

func TestVideoCommandBuildsVideoRequest(t *testing.T) {
	t.Parallel()

	fetcher := &fakeFetcher{
		detail:  &javdbapi.VideoDetail{Summary: javdbapi.VideoSummary{ID: "ZNdEbV", Code: "ZND-001"}},
		reviews: []javdbapi.Review{},
	}
	var stdout bytes.Buffer
	cmd := newCommand(stubBuilder(fetcher), &stdout, io.Discard)

	err := cmd.Run(context.Background(), []string{
		"javdbapi",
		"video",
		"--id", "ZNdEbV",
		"--output", "console",
	})
	require.NoError(t, err)
	require.Len(t, fetcher.detailCalls, 1)
	assert.Equal(t, javdbapi.VideoID("ZNdEbV"), fetcher.detailCalls[0])
	assert.Contains(t, stdout.String(), "ZND-001")
}

func TestVideoCommandRejectsInvalidID(t *testing.T) {
	t.Parallel()

	cmd := newCommand(stubBuilder(&fakeFetcher{}), io.Discard, io.Discard)
	err := cmd.Run(context.Background(), []string{"javdbapi", "video", "--id", ""})
	require.Error(t, err)
}

func TestSearchCommandDoesNotLogDoneOnZeroSummaryError(t *testing.T) {
	t.Parallel()

	fetcher := &fakeFetcher{pageErr: assert.AnError}
	var stderr bytes.Buffer
	cmd := newCommand(stubBuilder(fetcher), io.Discard, &stderr)

	err := cmd.Run(context.Background(), []string{
		"javdbapi",
		"search",
		"--keyword", "VR",
	})
	require.ErrorIs(t, err, assert.AnError)
	assert.NotContains(t, stderr.String(), "msg=done")
}

func TestConcurrencyFlagRejectsOutOfRangeValues(t *testing.T) {
	t.Parallel()

	cases := []int{0, 17}
	for _, c := range cases {
		cmd := newCommand(stubBuilder(&fakeFetcher{}), io.Discard, io.Discard)
		err := cmd.Run(context.Background(), []string{
			"javdbapi",
			"search",
			"--keyword", "VR",
			"--concurrency", strconv.Itoa(c),
		})
		require.Error(t, err)
		assert.Equal(t, "--concurrency must be between 1 and 16", err.Error())
	}
}

func TestRateFlagRejectsNegativeValue(t *testing.T) {
	t.Parallel()

	cmd := newCommand(buildRealFetcher, io.Discard, io.Discard)
	err := cmd.Run(context.Background(), []string{
		"javdbapi",
		"video",
		"--id", "ZNdEbV",
		"--rate", "-1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RequestsPerSecond must be > 0")
}

func TestRootVersionFlagPrintsBuildMetadata(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	t.Cleanup(func() {
		version = oldVersion
		commit = oldCommit
		date = oldDate
	})

	version = "v0.2.1"
	commit = "abc1234"
	date = "2026-04-19T09:00:00Z"

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := newCommand(stubBuilder(&fakeFetcher{}), &stdout, &stderr)

	err := cmd.Run(context.Background(), []string{"javdbapi", "--version"})
	require.NoError(t, err)
	assert.Equal(
		t,
		"javdbapi v0.2.1 (commit abc1234, built 2026-04-19T09:00:00Z)\n",
		stdout.String(),
	)
	assert.Empty(t, stderr.String())
}

func TestVersionCommandPrintsBuildMetadata(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	t.Cleanup(func() {
		version = oldVersion
		commit = oldCommit
		date = oldDate
	})

	version = "v0.2.1"
	commit = "abc1234"
	date = "2026-04-19T09:00:00Z"

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := newCommand(stubBuilder(&fakeFetcher{}), &stdout, &stderr)

	err := cmd.Run(context.Background(), []string{"javdbapi", "version"})
	require.NoError(t, err)
	assert.Equal(
		t,
		"javdbapi v0.2.1 (commit abc1234, built 2026-04-19T09:00:00Z)\n",
		stdout.String(),
	)
	assert.Empty(t, stderr.String())
}

func TestRootHelpShowsUsageForDataCommands(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := newCommand(stubBuilder(&fakeFetcher{}), &stdout, io.Discard)

	err := cmd.Run(context.Background(), []string{"javdbapi", "--help"})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "search")
	assert.Contains(t, stdout.String(), "search videos by keyword")
	assert.Contains(t, stdout.String(), "home")
	assert.Contains(t, stdout.String(), "browse home page listings")
	assert.Contains(t, stdout.String(), "maker")
	assert.Contains(t, stdout.String(), "list videos from a maker")
	assert.Contains(t, stdout.String(), "actor")
	assert.Contains(t, stdout.String(), "list videos from an actor")
	assert.Contains(t, stdout.String(), "ranking")
	assert.Contains(t, stdout.String(), "fetch ranked videos")
	assert.Contains(t, stdout.String(), "video")
	assert.Contains(t, stdout.String(), "fetch full video detail")
}

func TestSubcommandHelpShowsReadableUsageValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		args []string
		want []string
	}{
		{
			args: []string{"javdbapi", "home", "--help"},
			want: []string{
				"browse home page listings",
				"output mode (file|console|both) (default: file)",
				"home type (all|censored|uncensored|western)",
				"home filter (all|download|cnsub|review)",
				"home sort (publish|magnet)",
			},
		},
		{
			args: []string{"javdbapi", "maker", "--help"},
			want: []string{
				"list videos from a maker",
				"maker filter (all|playable|single|download|cnsub|preview)",
				"javdbapi maker --id 7R --filter playable",
			},
		},
		{
			args: []string{"javdbapi", "actor", "--help"},
			want: []string{
				"list videos from an actor",
				"actor filter (all|playable|single|download|cnsub)",
				"javdbapi actor --id neRNX --filter cnsub,download",
			},
		},
		{
			args: []string{"javdbapi", "ranking", "--help"},
			want: []string{
				"fetch ranked videos",
				"ranking period (daily|weekly|monthly)",
				"ranking type (censored|uncensored|western)",
				"javdbapi ranking --period weekly --type censored",
			},
		},
		{
			args: []string{"javdbapi", "video", "--help"},
			want: []string{
				"fetch full video detail",
				"output mode (file|console|both) (default: file)",
				"video ID, e.g. ZNdEbV",
				"javdbapi video --id ZNdEbV",
			},
		},
	}

	for _, tc := range cases {
		var stdout bytes.Buffer
		cmd := newCommand(stubBuilder(&fakeFetcher{}), &stdout, io.Discard)
		err := cmd.Run(context.Background(), tc.args)
		require.NoError(t, err)
		for _, want := range tc.want {
			assert.Contains(t, stdout.String(), want)
		}
	}
}

func TestSharedOutputHelpShowsSingleDefaultValue(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := newCommand(stubBuilder(&fakeFetcher{}), &stdout, io.Discard)

	err := cmd.Run(context.Background(), []string{"javdbapi", "home", "--help"})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "output mode (file|console|both) (default: file)")
	assert.NotContains(t, stdout.String(), `(default: file) (default: "file")`)
}

func TestHomeCommandNormalizesReadableDefaultsToOmittedWireValues(t *testing.T) {
	t.Parallel()

	fetcher := &fakeFetcher{}
	cmd := newCommand(stubBuilder(fetcher), io.Discard, io.Discard)

	err := cmd.Run(context.Background(), []string{
		"javdbapi",
		"home",
		"--type", "all",
		"--filter", "all",
		"--sort", "publish",
	})
	require.NoError(t, err)
	require.Len(t, fetcher.homeCalls, 1)
	assert.Equal(t, javdbapi.HomeType(""), fetcher.homeCalls[0].Type)
	assert.Equal(t, javdbapi.HomeFilter(""), fetcher.homeCalls[0].Filter)
	assert.Equal(t, javdbapi.HomeSort(""), fetcher.homeCalls[0].Sort)
}

func TestActorCommandAcceptsLegacyAliasesButStoresNormalizedRequestValues(t *testing.T) {
	t.Parallel()

	fetcher := &fakeFetcher{}
	cmd := newCommand(stubBuilder(fetcher), io.Discard, io.Discard)

	err := cmd.Run(context.Background(), []string{
		"javdbapi",
		"actor",
		"--id", "neRNX",
		"--filter", "c,d",
	})
	require.NoError(t, err)
	require.Len(t, fetcher.actorCalls, 1)
	assert.Equal(t, javdbapi.ActorID("neRNX"), fetcher.actorCalls[0].ActorID)
	assert.Equal(t, []javdbapi.ActorFilter{"c", "d"}, fetcher.actorCalls[0].Filters)
}

func TestMakerCommandParsesTypedMakerID(t *testing.T) {
	t.Parallel()

	fetcher := &fakeFetcher{}
	cmd := newCommand(stubBuilder(fetcher), io.Discard, io.Discard)

	err := cmd.Run(context.Background(), []string{
		"javdbapi",
		"maker",
		"--id", "7R",
		"--filter", "playable",
	})
	require.NoError(t, err)
	require.Len(t, fetcher.makerCalls, 1)
	assert.Equal(t, javdbapi.MakerID("7R"), fetcher.makerCalls[0].MakerID)
	assert.Equal(t, javdbapi.MakerFilter("playable"), fetcher.makerCalls[0].Filter)
}

func TestCommandValidationRejectsInvalidReadableEnums(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "home legacy numeric filter",
			args:    []string{"javdbapi", "home", "--filter", "1"},
			wantErr: `invalid --filter "1": must be one of all, download, cnsub, review`,
		},
		{
			name:    "actor all cannot be combined",
			args:    []string{"javdbapi", "actor", "--id", "neRNX", "--filter", "all,download"},
			wantErr: `invalid --filter "all,download": all cannot be combined with other values`,
		},
		{
			name:    "ranking period enum",
			args:    []string{"javdbapi", "ranking", "--period", "yearly", "--type", "censored"},
			wantErr: `invalid --period "yearly": must be one of daily, weekly, monthly`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newCommand(stubBuilder(&fakeFetcher{}), io.Discard, io.Discard)
			err := cmd.Run(context.Background(), tc.args)
			require.Error(t, err)
			assert.Equal(t, tc.wantErr, err.Error())
		})
	}
}

func TestVersionCommandPrintsDefaultBuildMetadata(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	t.Cleanup(func() {
		version = oldVersion
		commit = oldCommit
		date = oldDate
	})

	version = "dev"
	commit = "none"
	date = "unknown"

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := newCommand(stubBuilder(&fakeFetcher{}), &stdout, &stderr)

	err := cmd.Run(context.Background(), []string{"javdbapi", "version"})
	require.NoError(t, err)
	assert.Equal(
		t,
		"javdbapi dev (commit none, built unknown)\n",
		stdout.String(),
	)
	assert.Empty(t, stderr.String())
}
