package clioutput

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathKeyFromVideoPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{name: "plain_video_path", input: "/v/ZNdEbV", want: "ZNdEbV"},
		{name: "path_with_query", input: "/v/ZNdEbV?locale=zh#top", want: "ZNdEbV"},
		{name: "empty_path", input: "", wantErr: "missing video path"},
		{name: "non_video_path", input: "/actors/neRNX", wantErr: "invalid video path"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := PathKeyFromVideoPath(tc.input)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestSourceBuildersMarshalStableQueries(t *testing.T) {
	t.Parallel()

	actor := NewActorSource("neRNX", []string{"d", "", "c", "d"}, 2)
	raw, err := json.Marshal(actor)
	require.NoError(t, err)
	assert.Equal(
		t,
		`{"command":"actor","query":{"id":"neRNX","filter":["c","d"],"page":2}}`,
		string(raw),
	)

	video := NewVideoSource("/v/ZNdEbV")
	raw, err = json.Marshal(video)
	require.NoError(t, err)
	assert.Equal(
		t,
		`{"command":"video","query":{"path":"/v/ZNdEbV"}}`,
		string(raw),
	)
}
