package main

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	javdbapi "github.com/RPbro/javdbapi"
	"github.com/RPbro/javdbapi/internal/cliapp"
)

type fakeExecutor struct {
	listReq      cliapp.ListRequest
	videoReq     cliapp.VideoRequest
	listSummary  cliapp.Summary
	videoSummary cliapp.Summary
	listErr      error
	videoErr     error
}

func (f *fakeExecutor) RunList(_ context.Context, req cliapp.ListRequest) (cliapp.Summary, error) {
	f.listReq = req
	return f.listSummary, f.listErr
}

func (f *fakeExecutor) RunVideo(_ context.Context, req cliapp.VideoRequest) (cliapp.Summary, error) {
	f.videoReq = req
	return f.videoSummary, f.videoErr
}

func TestSearchCommandBuildsExpectedRequest(t *testing.T) {
	t.Parallel()

	executor := &fakeExecutor{}
	cmd := newCommand(executor, io.Discard, io.Discard)

	err := cmd.Run(context.Background(), []string{
		"javdbapi",
		"search",
		"--keyword", "VR",
		"--page", "2",
		"--max-pages", "3",
		"--output", "both",
		"--output-dir", "./custom-output",
		"--stale-after", "48h",
		"--timeout", "45s",
	})
	require.NoError(t, err)
	assert.Equal(t, cliapp.CommandSearch, executor.listReq.Command)
	assert.Equal(t, 2, executor.listReq.Page)
	assert.Equal(t, 3, executor.listReq.MaxPages)
	assert.Equal(t, cliapp.OutputBoth, executor.listReq.Shared.OutputMode)
	assert.Equal(t, "./custom-output", executor.listReq.Shared.OutputDir)
	assert.Equal(t, 48*time.Hour, executor.listReq.Shared.StaleAfter)
	assert.NotNil(t, executor.listReq.Search)
	assert.Equal(t, javdbapi.SearchQuery{Keyword: "VR"}, *executor.listReq.Search)
}

func TestVideoCommandRejectsPaginationFlags(t *testing.T) {
	t.Parallel()

	cmd := newCommand(&fakeExecutor{}, io.Discard, io.Discard)
	err := cmd.Run(context.Background(), []string{
		"javdbapi",
		"video",
		"--path", "/v/ZNdEbV",
		"--page", "2",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "flag provided but not defined")
}

func TestVideoCommandBuildsVideoRequest(t *testing.T) {
	t.Parallel()

	executor := &fakeExecutor{}
	var stdout bytes.Buffer
	cmd := newCommand(executor, &stdout, io.Discard)

	err := cmd.Run(context.Background(), []string{
		"javdbapi",
		"video",
		"--url", "https://javdb.com/v/ZNdEbV",
		"--output", "console",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://javdb.com/v/ZNdEbV", executor.videoReq.URL)
	assert.Equal(t, cliapp.OutputConsole, executor.videoReq.Shared.OutputMode)
}

func TestSearchCommandDoesNotLogDoneOnZeroSummaryError(t *testing.T) {
	t.Parallel()

	executor := &fakeExecutor{listErr: assert.AnError}
	var stderr bytes.Buffer
	cmd := newCommand(executor, io.Discard, &stderr)

	err := cmd.Run(context.Background(), []string{
		"javdbapi",
		"search",
		"--keyword", "VR",
	})
	require.ErrorIs(t, err, assert.AnError)
	assert.NotContains(t, stderr.String(), "msg=done")
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
	cmd := newCommand(&fakeExecutor{}, &stdout, &stderr)

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
	cmd := newCommand(&fakeExecutor{}, &stdout, &stderr)

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
	cmd := newCommand(&fakeExecutor{}, &stdout, io.Discard)

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
				"video path, e.g. /v/ZNdEbV",
				"full video URL; host must match --base-url",
			},
		},
	}

	for _, tc := range cases {
		var stdout bytes.Buffer
		cmd := newCommand(&fakeExecutor{}, &stdout, io.Discard)
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
	cmd := newCommand(&fakeExecutor{}, &stdout, io.Discard)

	err := cmd.Run(context.Background(), []string{"javdbapi", "home", "--help"})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "output mode (file|console|both) (default: file)")
	assert.NotContains(t, stdout.String(), `(default: file) (default: "file")`)
}

func TestHomeCommandNormalizesReadableDefaultsToOmittedWireValues(t *testing.T) {
	t.Parallel()

	executor := &fakeExecutor{}
	cmd := newCommand(executor, io.Discard, io.Discard)

	err := cmd.Run(context.Background(), []string{
		"javdbapi",
		"home",
		"--type", "all",
		"--filter", "all",
		"--sort", "publish",
	})
	require.NoError(t, err)
	require.NotNil(t, executor.listReq.Home)
	assert.Equal(t, javdbapi.HomeType(""), executor.listReq.Home.Type)
	assert.Equal(t, javdbapi.HomeFilter(""), executor.listReq.Home.Filter)
	assert.Equal(t, javdbapi.HomeSort(""), executor.listReq.Home.Sort)
}

func TestActorCommandAcceptsLegacyAliasesButStoresNormalizedRequestValues(t *testing.T) {
	t.Parallel()

	executor := &fakeExecutor{}
	cmd := newCommand(executor, io.Discard, io.Discard)

	err := cmd.Run(context.Background(), []string{
		"javdbapi",
		"actor",
		"--id", "neRNX",
		"--filter", "c,d",
	})
	require.NoError(t, err)
	require.NotNil(t, executor.listReq.Actor)
	assert.Equal(t, []javdbapi.ActorFilter{"c", "d"}, executor.listReq.Actor.Filters)
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
			cmd := newCommand(&fakeExecutor{}, io.Discard, io.Discard)
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
	cmd := newCommand(&fakeExecutor{}, &stdout, &stderr)

	err := cmd.Run(context.Background(), []string{"javdbapi", "version"})
	require.NoError(t, err)
	assert.Equal(
		t,
		"javdbapi dev (commit none, built unknown)\n",
		stdout.String(),
	)
	assert.Empty(t, stderr.String())
}
