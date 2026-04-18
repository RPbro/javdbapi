package clioutput

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	javdbapi "github.com/RPbro/javdbapi"
)

func TestStoreLoadHandlesFreshStaleAndBrokenFiles(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.FixedZone("CST", 8*3600))
	dir := t.TempDir()
	store := NewStore(dir, func() time.Time { return now })

	state, err := store.Load("/v/missing", 24*time.Hour)
	require.NoError(t, err)
	assert.False(t, state.Exists)
	assert.False(t, state.Fresh)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{"), 0o644))
	state, err = store.Load("/v/broken", 24*time.Hour)
	require.NoError(t, err)
	assert.True(t, state.Exists)
	assert.False(t, state.Fresh)
	assert.Nil(t, state.Document)

	fresh := Document{
		Metadata: Metadata{
			LastUpdated: now.Add(-1 * time.Hour),
			Path:        "/v/fresh",
			PathKey:     "fresh",
			Sources:     []Source{NewSearchSource("VR", 1)},
		},
		Video: javdbapi.Video{ID: "/v/fresh", Code: "FRESH-001"},
	}
	require.NoError(t, store.WriteFile(fresh))

	state, err = store.Load("/v/fresh", 24*time.Hour)
	require.NoError(t, err)
	require.NotNil(t, state.Document)
	assert.True(t, state.Exists)
	assert.True(t, state.Fresh)
	assert.Equal(t, "/v/fresh", state.Document.Metadata.Path)
}

func TestStoreWriteFileMergesSourcesWithoutDuplicates(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	store := NewStore(dir, func() time.Time { return now })

	existing := Document{
		Metadata: Metadata{
			LastUpdated: now.Add(-48 * time.Hour),
			Path:        "/v/ZNdEbV",
			PathKey:     "ZNdEbV",
			Sources:     []Source{NewSearchSource("VR", 1)},
		},
		Video: javdbapi.Video{ID: "/v/ZNdEbV", Code: "ABC-123"},
	}
	require.NoError(t, store.WriteFile(existing))

	next := Document{
		Metadata: Metadata{
			LastUpdated: now,
			Path:        "/v/ZNdEbV",
			PathKey:     "ZNdEbV",
			Sources: []Source{
				NewSearchSource("VR", 1),
				NewVideoSource("/v/ZNdEbV"),
			},
		},
		Video: javdbapi.Video{ID: "/v/ZNdEbV", Code: "ABC-123"},
	}
	require.NoError(t, store.WriteFile(next))

	loaded, err := store.Load("/v/ZNdEbV", 24*time.Hour)
	require.NoError(t, err)
	require.NotNil(t, loaded.Document)
	require.Len(t, loaded.Document.Metadata.Sources, 2)

	var stdout bytes.Buffer
	require.NoError(t, store.WriteJSON(&stdout, *loaded.Document))
	assert.Contains(t, stdout.String(), `"command":"video"`)
}
