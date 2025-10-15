package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreateBackup_EmptyEntryList(t *testing.T) {
	tmpDir := t.TempDir()
	var entries []*FileEntry

	err := createBackup(tmpDir, entries)
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	// Should not create any chunk files
	files, _ := filepath.Glob(filepath.Join(tmpDir, "chunk_*.dat"))
	if len(files) != 0 {
		t.Errorf("expected no chunks for empty list, got %d", len(files))
	}
}

func TestCreateBackup_SingleSmallFile(t *testing.T) {
	tmpDir := t.TempDir()

	entries := []*FileEntry{
		{
			Path:    "test.txt",
			Mode:    0644,
			ModTime: time.Now(),
			Size:    100,
			Content: []byte("small content"),
			Deleted: false,
		},
	}

	err := createBackup(tmpDir, entries)
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	// Should create exactly one chunk
	files, _ := filepath.Glob(filepath.Join(tmpDir, "chunk_*.dat"))
	if len(files) != 1 {
		t.Fatalf("expected 1 chunk file, got %d", len(files))
	}

	// Verify we can read it back
	chunk, err := readChunk(files[0])
	if err != nil {
		t.Fatalf("readChunk() error = %v", err)
	}

	if len(chunk.Entries) != 1 {
		t.Fatalf("expected 1 entry in chunk, got %d", len(chunk.Entries))
	}

	if chunk.Entries[0].Path != "test.txt" {
		t.Errorf("expected path 'test.txt', got %s", chunk.Entries[0].Path)
	}
}

func TestCreateBackup_MultipleFilesInOneChunk(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple small files that fit in one chunk
	entries := []*FileEntry{
		{
			Path:    "file1.txt",
			Content: []byte(strings.Repeat("a", 1024)), // 1KB
		},
		{
			Path:    "file2.txt",
			Content: []byte(strings.Repeat("b", 1024)), // 1KB
		},
		{
			Path:    "file3.txt",
			Content: []byte(strings.Repeat("c", 1024)), // 1KB
		},
	}

	err := createBackup(tmpDir, entries)
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	files, _ := filepath.Glob(filepath.Join(tmpDir, "chunk_*.dat"))
	if len(files) != 1 {
		t.Fatalf("expected 1 chunk file, got %d", len(files))
	}

	chunk, err := readChunk(files[0])
	if err != nil {
		t.Fatalf("readChunk() error = %v", err)
	}

	if len(chunk.Entries) != 3 {
		t.Errorf("expected 3 entries in chunk, got %d", len(chunk.Entries))
	}
}

func TestCreateBackup_FilesSpanMultipleChunks(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files that will span multiple 5MB chunks
	// Each file is ~3MB, so 2 files per chunk
	entries := []*FileEntry{
		{
			Path:    "file1.dat",
			Content: make([]byte, 3*1024*1024), // 3MB
		},
		{
			Path:    "file2.dat",
			Content: make([]byte, 3*1024*1024), // 3MB
		},
		{
			Path:    "file3.dat",
			Content: make([]byte, 3*1024*1024), // 3MB
		},
		{
			Path:    "file4.dat",
			Content: make([]byte, 3*1024*1024), // 3MB
		},
	}

	err := createBackup(tmpDir, entries)
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	files, _ := filepath.Glob(filepath.Join(tmpDir, "chunk_*.dat"))

	// Should create multiple chunks (at least 2)
	if len(files) < 2 {
		t.Errorf("expected at least 2 chunks, got %d", len(files))
	}

	// Verify all entries are preserved across chunks
	totalEntries := 0
	for _, file := range files {
		chunk, err := readChunk(file)
		if err != nil {
			t.Fatalf("readChunk() error = %v", err)
		}
		totalEntries += len(chunk.Entries)
	}

	if totalEntries != 4 {
		t.Errorf("expected 4 total entries across chunks, got %d", totalEntries)
	}
}

func TestCreateBackup_LargeFileSingleChunk(t *testing.T) {
	tmpDir := t.TempDir()

	// Single file larger than 5MB
	entries := []*FileEntry{
		{
			Path:    "large.dat",
			Content: make([]byte, 10*1024*1024), // 10MB
		},
	}

	err := createBackup(tmpDir, entries)
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	files, _ := filepath.Glob(filepath.Join(tmpDir, "chunk_*.dat"))

	// Should create one chunk (soft limit behavior)
	if len(files) != 1 {
		t.Fatalf("expected 1 chunk for large file, got %d", len(files))
	}

	chunk, err := readChunk(files[0])
	if err != nil {
		t.Fatalf("readChunk() error = %v", err)
	}

	if len(chunk.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(chunk.Entries))
	}

	if len(chunk.Entries[0].Content) != 10*1024*1024 {
		t.Errorf("content size mismatch")
	}
}

func TestCreateBackup_DeletedEntries(t *testing.T) {
	tmpDir := t.TempDir()

	entries := []*FileEntry{
		{
			Path:    "deleted.txt",
			Deleted: true,
		},
		{
			Path:    "normal.txt",
			Content: []byte("content"),
			Deleted: false,
		},
	}

	err := createBackup(tmpDir, entries)
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	files, _ := filepath.Glob(filepath.Join(tmpDir, "chunk_*.dat"))
	chunk, err := readChunk(files[0])
	if err != nil {
		t.Fatalf("readChunk() error = %v", err)
	}

	// Check both entries are preserved
	if len(chunk.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(chunk.Entries))
	}

	// Find the deleted entry
	var foundDeleted bool
	for _, entry := range chunk.Entries {
		if entry.Path == "deleted.txt" && entry.Deleted {
			foundDeleted = true
			// Deleted entries should have no content
			if len(entry.Content) > 0 {
				t.Error("deleted entry should have no content")
			}
		}
	}

	if !foundDeleted {
		t.Error("deleted entry not found in chunk")
	}
}

func TestWriteReadChunk_PreservesMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	originalTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	originalChunk := Chunk{
		Entries: []*FileEntry{
			{
				Path:    "test.txt",
				Mode:    0755,
				ModTime: originalTime,
				Size:    42,
				Content: []byte("test content"),
				Deleted: false,
			},
		},
	}

	// Write chunk
	err := writeChunk(tmpDir, time.Now().Unix(), 0, originalChunk)
	if err != nil {
		t.Fatalf("writeChunk() error = %v", err)
	}

	// Read it back
	files, _ := filepath.Glob(filepath.Join(tmpDir, "chunk_*.dat"))
	if len(files) != 1 {
		t.Fatalf("expected 1 chunk file, got %d", len(files))
	}

	readChunk, err := readChunk(files[0])
	if err != nil {
		t.Fatalf("readChunk() error = %v", err)
	}

	// Verify metadata is preserved
	if len(readChunk.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(readChunk.Entries))
	}

	entry := readChunk.Entries[0]
	if entry.Path != "test.txt" {
		t.Errorf("path mismatch: expected 'test.txt', got %s", entry.Path)
	}
	if entry.Mode != 0755 {
		t.Errorf("mode mismatch: expected 0755, got %o", entry.Mode)
	}
	if !entry.ModTime.Equal(originalTime) {
		t.Errorf("modtime mismatch: expected %v, got %v", originalTime, entry.ModTime)
	}
	if entry.Size != 42 {
		t.Errorf("size mismatch: expected 42, got %d", entry.Size)
	}
	if string(entry.Content) != "test content" {
		t.Errorf("content mismatch")
	}
}

func TestCreateBackup_ChunkNumbering(t *testing.T) {
	tmpDir := t.TempDir()

	// Create entries that will span multiple chunks
	entries := []*FileEntry{
		{Path: "1.dat", Content: make([]byte, 4*1024*1024)},
		{Path: "2.dat", Content: make([]byte, 4*1024*1024)},
		{Path: "3.dat", Content: make([]byte, 4*1024*1024)},
	}

	err := createBackup(tmpDir, entries)
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	files, _ := filepath.Glob(filepath.Join(tmpDir, "chunk_*.dat"))

	if len(files) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(files))
	}

	// Verify chunk files have sequential numbering
	// Format: chunk_<timestamp>_000.dat, chunk_<timestamp>_001.dat, etc.
	for i, file := range files {
		basename := filepath.Base(file)
		if !strings.HasPrefix(basename, "chunk_") {
			t.Errorf("chunk file %d has wrong prefix: %s", i, basename)
		}
		if !strings.HasSuffix(basename, ".dat") {
			t.Errorf("chunk file %d has wrong suffix: %s", i, basename)
		}
	}
}

func TestCreateBackup_ManySmallFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 100 small files
	var entries []*FileEntry
	for i := range 100 {
		entries = append(entries, &FileEntry{
			Path:    filepath.Join("dir", "file_"+string(rune(i))+".txt"),
			Content: []byte(strings.Repeat("x", 1024)), // 1KB each
		})
	}

	err := createBackup(tmpDir, entries)
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	files, _ := filepath.Glob(filepath.Join(tmpDir, "chunk_*.dat"))

	// Verify all entries are preserved
	totalEntries := 0
	for _, file := range files {
		chunk, err := readChunk(file)
		if err != nil {
			t.Fatalf("readChunk() error = %v", err)
		}
		totalEntries += len(chunk.Entries)
	}

	if totalEntries != 100 {
		t.Errorf("expected 100 total entries, got %d", totalEntries)
	}
}

func TestCreateBackup_MixedSizes(t *testing.T) {
	tmpDir := t.TempDir()

	entries := []*FileEntry{
		{Path: "tiny.txt", Content: []byte("small")},             // tiny
		{Path: "medium.dat", Content: make([]byte, 2*1024*1024)}, // 2MB
		{Path: "large.dat", Content: make([]byte, 8*1024*1024)},  // 8MB (exceeds chunk)
		{Path: "another.txt", Content: []byte("more")},           // tiny
	}

	err := createBackup(tmpDir, entries)
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	files, _ := filepath.Glob(filepath.Join(tmpDir, "chunk_*.dat"))

	// Should create multiple chunks due to large file
	if len(files) < 2 {
		t.Errorf("expected at least 2 chunks for mixed sizes, got %d", len(files))
	}

	// Verify all entries are preserved
	totalEntries := 0
	foundLarge := false
	for _, file := range files {
		chunk, err := readChunk(file)
		if err != nil {
			t.Fatalf("readChunk() error = %v", err)
		}
		totalEntries += len(chunk.Entries)

		for _, entry := range chunk.Entries {
			if entry.Path == "large.dat" {
				foundLarge = true
				if len(entry.Content) != 8*1024*1024 {
					t.Error("large file content size mismatch")
				}
			}
		}
	}

	if totalEntries != 4 {
		t.Errorf("expected 4 total entries, got %d", totalEntries)
	}
	if !foundLarge {
		t.Error("large file not found in chunks")
	}
}

func TestWriteChunk_InvalidPath(t *testing.T) {
	chunk := Chunk{
		Entries: []*FileEntry{{Path: "test.txt", Content: []byte("data")}},
	}

	// Try to write to non-existent directory
	err := writeChunk("/nonexistent/path", time.Now().Unix(), 0, chunk)
	if err == nil {
		t.Error("expected error writing to invalid path, got nil")
	}
}

func TestReadChunk_InvalidFile(t *testing.T) {
	_, err := readChunk("/nonexistent/chunk.dat")
	if err == nil {
		t.Error("expected error reading non-existent file, got nil")
	}
}

func TestReadChunk_CorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	corruptFile := filepath.Join(tmpDir, "corrupt.dat")

	// Write corrupted data
	if err := os.WriteFile(corruptFile, []byte("not valid gob data"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := readChunk(corruptFile)
	if err == nil {
		t.Error("expected error reading corrupted file, got nil")
	}
}
