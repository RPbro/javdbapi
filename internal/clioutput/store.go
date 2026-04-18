package clioutput

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type CacheState struct {
	FilePath string
	Exists   bool
	Fresh    bool
	Document *Document
}

type Store struct {
	outputDir string
	now       func() time.Time
}

func NewStore(outputDir string, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{
		outputDir: outputDir,
		now:       now,
	}
}

func (s *Store) Load(videoPath string, staleAfter time.Duration) (CacheState, error) {
	filePath, err := s.FilePath(videoPath)
	if err != nil {
		return CacheState{}, err
	}

	state := CacheState{FilePath: filePath}
	body, err := os.ReadFile(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return CacheState{}, fmt.Errorf("read cache file: %w", err)
	}

	state.Exists = true

	var doc Document
	if err := json.Unmarshal(body, &doc); err != nil {
		return state, nil
	}
	if doc.Metadata.Path == "" || doc.Metadata.LastUpdated.IsZero() {
		return state, nil
	}
	if doc.Metadata.Path != videoPath {
		return state, nil
	}

	state.Document = &doc
	state.Fresh = s.now().Sub(doc.Metadata.LastUpdated) <= staleAfter
	return state, nil
}

func (s *Store) FilePath(videoPath string) (string, error) {
	key, err := PathKeyFromVideoPath(videoPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(s.outputDir, key+".json"), nil
}

func (s *Store) WriteFile(doc Document) error {
	if err := os.MkdirAll(s.outputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	merged, err := s.mergeWithExisting(doc)
	if err != nil {
		return err
	}

	filePath, err := s.FilePath(merged.Metadata.Path)
	if err != nil {
		return err
	}

	body, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal document: %w", err)
	}
	body = append(body, '\n')

	if err := os.WriteFile(filePath, body, 0o644); err != nil {
		return fmt.Errorf("write cache file: %w", err)
	}
	return nil
}

func (s *Store) WriteJSON(w io.Writer, doc Document) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(doc)
}

func MergeSources(existing []Source, next []Source) []Source {
	merged := make([]Source, 0, len(existing)+len(next))
	seen := make(map[string]struct{}, len(existing)+len(next))

	for _, source := range append(append([]Source{}, existing...), next...) {
		key := source.Key()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, source)
	}

	return merged
}

func (s *Store) mergeWithExisting(doc Document) (Document, error) {
	state, err := s.Load(doc.Metadata.Path, 0)
	if err != nil {
		return Document{}, err
	}
	if state.Document == nil {
		return doc, nil
	}

	doc.Metadata.Sources = MergeSources(state.Document.Metadata.Sources, doc.Metadata.Sources)
	return doc, nil
}
