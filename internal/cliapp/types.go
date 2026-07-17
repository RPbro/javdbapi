package cliapp

import (
	"io"
	"log/slog"
	"time"

	javdbapi "github.com/RPbro/javdbapi"
)

type CommandName string

const (
	CommandHome    CommandName = "home"
	CommandSearch  CommandName = "search"
	CommandMaker   CommandName = "maker"
	CommandActor   CommandName = "actor"
	CommandRanking CommandName = "ranking"
)

type OutputMode string

const (
	OutputConsole OutputMode = "console"
	OutputFile    OutputMode = "file"
	OutputBoth    OutputMode = "both"
)

// SharedOptions carries execution options into the cliapp Run functions. SDK
// construction parameters (base URL, proxy, user agent, timeout, rate/retry
// policy) live entirely in cmd/javdbapi, since cliapp only ever talks to a
// Fetcher and never builds a javdbapi.Client itself.
type SharedOptions struct {
	OutputMode  OutputMode
	OutputDir   string
	StaleAfter  time.Duration
	Concurrency int
	FailFast    bool
	Stdout      io.Writer
	Logger      *slog.Logger
}

type ListRequest struct {
	Shared      SharedOptions
	Command     CommandName
	Page        int
	MaxPages    int
	SummaryOnly bool
	Home        *javdbapi.HomeQuery
	Search      *javdbapi.SearchQuery
	Maker       *javdbapi.MakerVideosQuery
	Actor       *javdbapi.ActorVideosQuery
	Ranking     *javdbapi.RankingQuery
}

type VideoRequest struct {
	Shared SharedOptions
	ID     javdbapi.VideoID
}

type VideoRef struct {
	ID      javdbapi.VideoID
	Title   string
	Code    string
	Page    int
	Summary javdbapi.VideoSummary
}

// Summary reports command-wide outcomes. PartialFailed counts videos whose
// Detail persisted successfully but whose Reviews fetch failed; it is never
// conflated with Failed, which counts videos that produced no usable output.
type Summary struct {
	PagesScanned    int
	Candidates      int
	Deduplicated    int
	Fetched         int
	SkippedFresh    int
	Failed          int
	PartialFailed   int
	SummariesOutput int
}
