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
