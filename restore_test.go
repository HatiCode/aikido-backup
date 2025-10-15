package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRestore_NoChunksFound(t *testing.T) {
	tmpBackup := t.TempDir()
	tmpRestore := t.TempDir()

	err := restore(tmpBackup, tmpRestore)
	if err == nil {
		t.Error("expected error when no chunks found, got nil")
	}
}

func TestRestore_SingleFile(t *testing.T) {
	tmpBackup := t.TempDir()
	tmpRestore := t.TempDir()

	// Create a chunk with one file
	chunk := Chunk{
		Entries: []*FileEntry{
			{
				Path:    "test.txt",
				Mode:    0644,
				ModTime: time.Now(),
				Size:    12,
				Content: []byte("test content"),
				Deleted: false,
			},
		},
	}

	if err := writeChunk(tmpBackup, 1000, 0, chunk); err != nil {
		t.Fatal(err)
	}

	// Restore
	err := restore(tmpBackup, tmpRestore)
	if err != nil {
		t.Fatalf("restore() error = %v", err)
	}

	// Verify file exists
	restoredFile := filepath.Join(tmpRestore, "test.txt")
	if _, err := os.Stat(restoredFile); os.IsNotExist(err) {
		t.Error("restored file does not exist")
	}

	// Verify content
	content, err := os.ReadFile(restoredFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "test content" {
		t.Errorf("content mismatch: expected 'test content', got %s", string(content))
	}
}

func TestRestore_MultipleFiles(t *testing.T) {
	tmpBackup := t.TempDir()
	tmpRestore := t.TempDir()

	chunk := Chunk{
		Entries: []*FileEntry{
			{
				Path:    "file1.txt",
				Mode:    0644,
				Content: []byte("content1"),
			},
			{
				Path:    "file2.txt",
				Mode:    0644,
				Content: []byte("content2"),
			},
			{
				Path:    "file3.txt",
				Mode:    0644,
				Content: []byte("content3"),
			},
		},
	}

	if err := writeChunk(tmpBackup, 1000, 0, chunk); err != nil {
		t.Fatal(err)
	}

	err := restore(tmpBackup, tmpRestore)
	if err != nil {
		t.Fatalf("restore() error = %v", err)
	}

	// Verify all files exist
	for i := 1; i <= 3; i++ {
		filename := filepath.Join(tmpRestore, "file"+string(rune('0'+i))+".txt")
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			t.Errorf("file%d.txt does not exist", i)
		}
	}
}

func TestRestore_NestedDirectoryStructure(t *testing.T) {
	tmpBackup := t.TempDir()
	tmpRestore := t.TempDir()

	chunk := Chunk{
		Entries: []*FileEntry{
			{
				Path:    filepath.Join("dir1", "file1.txt"),
				Mode:    0644,
				Content: []byte("content1"),
			},
			{
				Path:    filepath.Join("dir1", "subdir", "file2.txt"),
				Mode:    0644,
				Content: []byte("content2"),
			},
			{
				Path:    filepath.Join("dir2", "file3.txt"),
				Mode:    0644,
				Content: []byte("content3"),
			},
		},
	}

	if err := writeChunk(tmpBackup, 1000, 0, chunk); err != nil {
		t.Fatal(err)
	}

	err := restore(tmpBackup, tmpRestore)
	if err != nil {
		t.Fatalf("restore() error = %v", err)
	}

	// Verify nested structure exists
	testCases := []string{
		filepath.Join("dir1", "file1.txt"),
		filepath.Join("dir1", "subdir", "file2.txt"),
		filepath.Join("dir2", "file3.txt"),
	}

	for _, tc := range testCases {
		fullPath := filepath.Join(tmpRestore, tc)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("nested file does not exist: %s", tc)
		}
	}
}

func TestRestore_PreservesPermissions(t *testing.T) {
	tmpBackup := t.TempDir()
	tmpRestore := t.TempDir()

	chunk := Chunk{
		Entries: []*FileEntry{
			{
				Path:    "executable.sh",
				Mode:    0755,
				Content: []byte("#!/bin/bash\necho test"),
			},
			{
				Path:    "readonly.txt",
				Mode:    0444,
				Content: []byte("readonly"),
			},
		},
	}

	if err := writeChunk(tmpBackup, 1000, 0, chunk); err != nil {
		t.Fatal(err)
	}

	err := restore(tmpBackup, tmpRestore)
	if err != nil {
		t.Fatalf("restore() error = %v", err)
	}

	// Check executable
	execFile := filepath.Join(tmpRestore, "executable.sh")
	info, err := os.Stat(execFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0755 {
		t.Errorf("executable permissions: expected 0755, got %o", info.Mode().Perm())
	}

	// Check readonly
	readonlyFile := filepath.Join(tmpRestore, "readonly.txt")
	info, err = os.Stat(readonlyFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf("readonly permissions: expected 0444, got %o", info.Mode().Perm())
	}
}

func TestRestore_LaterChunkOverridesEarlier(t *testing.T) {
	tmpBackup := t.TempDir()
	tmpRestore := t.TempDir()

	// First chunk - original content
	chunk1 := Chunk{
		Entries: []*FileEntry{
			{
				Path:    "file.txt",
				Mode:    0644,
				Content: []byte("original content"),
			},
		},
	}

	// Second chunk - modified content (later timestamp)
	chunk2 := Chunk{
		Entries: []*FileEntry{
			{
				Path:    "file.txt",
				Mode:    0644,
				Content: []byte("modified content"),
			},
		},
	}

	// Write chunks with different timestamps
	if err := writeChunk(tmpBackup, 1000, 0, chunk1); err != nil {
		t.Fatal(err)
	}
	if err := writeChunk(tmpBackup, 2000, 0, chunk2); err != nil {
		t.Fatal(err)
	}

	err := restore(tmpBackup, tmpRestore)
	if err != nil {
		t.Fatalf("restore() error = %v", err)
	}

	// Should have the LATER content
	restoredFile := filepath.Join(tmpRestore, "file.txt")
	content, err := os.ReadFile(restoredFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "modified content" {
		t.Errorf("expected 'modified content', got %s", string(content))
	}
}

func TestRestore_FileDeletedInLaterChunk(t *testing.T) {
	tmpBackup := t.TempDir()
	tmpRestore := t.TempDir()

	// First chunk - file exists
	chunk1 := Chunk{
		Entries: []*FileEntry{
			{
				Path:    "file.txt",
				Mode:    0644,
				Content: []byte("content"),
			},
		},
	}

	// Second chunk - file deleted
	chunk2 := Chunk{
		Entries: []*FileEntry{
			{
				Path:    "file.txt",
				Deleted: true,
			},
		},
	}

	if err := writeChunk(tmpBackup, 1000, 0, chunk1); err != nil {
		t.Fatal(err)
	}
	if err := writeChunk(tmpBackup, 2000, 0, chunk2); err != nil {
		t.Fatal(err)
	}

	err := restore(tmpBackup, tmpRestore)
	if err != nil {
		t.Fatalf("restore() error = %v", err)
	}

	// File should NOT be restored
	restoredFile := filepath.Join(tmpRestore, "file.txt")
	if _, err := os.Stat(restoredFile); !os.IsNotExist(err) {
		t.Error("deleted file should not be restored")
	}
}

func TestRestore_FileDeletedThenRecreated(t *testing.T) {
	tmpBackup := t.TempDir()
	tmpRestore := t.TempDir()

	// Chunk 1 - file exists
	chunk1 := Chunk{
		Entries: []*FileEntry{
			{
				Path:    "file.txt",
				Mode:    0644,
				Content: []byte("original"),
			},
		},
	}

	// Chunk 2 - file deleted
	chunk2 := Chunk{
		Entries: []*FileEntry{
			{
				Path:    "file.txt",
				Deleted: true,
			},
		},
	}

	// Chunk 3 - file recreated with new content
	chunk3 := Chunk{
		Entries: []*FileEntry{
			{
				Path:    "file.txt",
				Mode:    0644,
				Content: []byte("recreated"),
			},
		},
	}

	if err := writeChunk(tmpBackup, 1000, 0, chunk1); err != nil {
		t.Fatal(err)
	}
	if err := writeChunk(tmpBackup, 2000, 0, chunk2); err != nil {
		t.Fatal(err)
	}
	if err := writeChunk(tmpBackup, 3000, 0, chunk3); err != nil {
		t.Fatal(err)
	}

	err := restore(tmpBackup, tmpRestore)
	if err != nil {
		t.Fatalf("restore() error = %v", err)
	}

	// Should have the recreated version
	restoredFile := filepath.Join(tmpRestore, "file.txt")
	content, err := os.ReadFile(restoredFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "recreated" {
		t.Errorf("expected 'recreated', got %s", string(content))
	}
}

func TestRestore_OnlyDeletedEntries(t *testing.T) {
	tmpBackup := t.TempDir()
	tmpRestore := t.TempDir()

	chunk := Chunk{
		Entries: []*FileEntry{
			{
				Path:    "file1.txt",
				Deleted: true,
			},
			{
				Path:    "file2.txt",
				Deleted: true,
			},
		},
	}

	if err := writeChunk(tmpBackup, 1000, 0, chunk); err != nil {
		t.Fatal(err)
	}

	err := restore(tmpBackup, tmpRestore)
	if err != nil {
		t.Fatalf("restore() error = %v", err)
	}

	// No files should be restored
	entries, err := os.ReadDir(tmpRestore)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 0 {
		t.Errorf("expected no files restored, got %d", len(entries))
	}
}

func TestRestore_MixedDeletedAndNormal(t *testing.T) {
	tmpBackup := t.TempDir()
	tmpRestore := t.TempDir()

	chunk := Chunk{
		Entries: []*FileEntry{
			{
				Path:    "keep.txt",
				Mode:    0644,
				Content: []byte("keep this"),
			},
			{
				Path:    "delete.txt",
				Deleted: true,
			},
			{
				Path:    "also_keep.txt",
				Mode:    0644,
				Content: []byte("also keep"),
			},
		},
	}

	if err := writeChunk(tmpBackup, 1000, 0, chunk); err != nil {
		t.Fatal(err)
	}

	err := restore(tmpBackup, tmpRestore)
	if err != nil {
		t.Fatalf("restore() error = %v", err)
	}

	// Check kept files exist
	if _, err := os.Stat(filepath.Join(tmpRestore, "keep.txt")); os.IsNotExist(err) {
		t.Error("keep.txt should exist")
	}
	if _, err := os.Stat(filepath.Join(tmpRestore, "also_keep.txt")); os.IsNotExist(err) {
		t.Error("also_keep.txt should exist")
	}

	// Check deleted file doesn't exist
	if _, err := os.Stat(filepath.Join(tmpRestore, "delete.txt")); !os.IsNotExist(err) {
		t.Error("delete.txt should not exist")
	}
}

func TestRestore_MultipleBackupRuns(t *testing.T) {
	tmpBackup := t.TempDir()
	tmpRestore := t.TempDir()

	// Simulate multiple backup runs at different times
	// Backup 1 at timestamp 1000
	chunk1 := Chunk{
		Entries: []*FileEntry{
			{Path: "file1.txt", Mode: 0644, Content: []byte("v1")},
			{Path: "file2.txt", Mode: 0644, Content: []byte("v1")},
		},
	}

	// Backup 2 at timestamp 2000
	chunk2 := Chunk{
		Entries: []*FileEntry{
			{Path: "file1.txt", Mode: 0644, Content: []byte("v2")}, // modified
			{Path: "file3.txt", Mode: 0644, Content: []byte("v1")}, // new
		},
	}

	// Backup 3 at timestamp 3000
	chunk3 := Chunk{
		Entries: []*FileEntry{
			{Path: "file2.txt", Deleted: true},                     // deleted
			{Path: "file3.txt", Mode: 0644, Content: []byte("v2")}, // modified
		},
	}

	if err := writeChunk(tmpBackup, 1000, 0, chunk1); err != nil {
		t.Fatal(err)
	}
	if err := writeChunk(tmpBackup, 2000, 0, chunk2); err != nil {
		t.Fatal(err)
	}
	if err := writeChunk(tmpBackup, 3000, 0, chunk3); err != nil {
		t.Fatal(err)
	}

	err := restore(tmpBackup, tmpRestore)
	if err != nil {
		t.Fatalf("restore() error = %v", err)
	}

	// file1.txt should have v2
	content, _ := os.ReadFile(filepath.Join(tmpRestore, "file1.txt"))
	if string(content) != "v2" {
		t.Errorf("file1.txt: expected 'v2', got %s", string(content))
	}

	// file2.txt should not exist (deleted)
	if _, err := os.Stat(filepath.Join(tmpRestore, "file2.txt")); !os.IsNotExist(err) {
		t.Error("file2.txt should not exist")
	}

	// file3.txt should have v2
	content, _ = os.ReadFile(filepath.Join(tmpRestore, "file3.txt"))
	if string(content) != "v2" {
		t.Errorf("file3.txt: expected 'v2', got %s", string(content))
	}
}

func TestRestore_ChunksProcessedInOrder(t *testing.T) {
	tmpBackup := t.TempDir()
	tmpRestore := t.TempDir()

	// Create chunks with different timestamps in non-sequential order
	// (to test that alphabetical sort gives chronological order)

	chunk1 := Chunk{
		Entries: []*FileEntry{
			{Path: "file.txt", Mode: 0644, Content: []byte("first")},
		},
	}
	chunk2 := Chunk{
		Entries: []*FileEntry{
			{Path: "file.txt", Mode: 0644, Content: []byte("second")},
		},
	}
	chunk3 := Chunk{
		Entries: []*FileEntry{
			{Path: "file.txt", Mode: 0644, Content: []byte("third")},
		},
	}

	// Write in order
	if err := writeChunk(tmpBackup, 1000, 0, chunk1); err != nil {
		t.Fatal(err)
	}
	if err := writeChunk(tmpBackup, 2000, 0, chunk2); err != nil {
		t.Fatal(err)
	}
	if err := writeChunk(tmpBackup, 3000, 0, chunk3); err != nil {
		t.Fatal(err)
	}

	err := restore(tmpBackup, tmpRestore)
	if err != nil {
		t.Fatalf("restore() error = %v", err)
	}

	// Should have the LAST content
	content, err := os.ReadFile(filepath.Join(tmpRestore, "file.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "third" {
		t.Errorf("expected 'third', got %s", string(content))
	}
}

func TestRestore_InvalidBackupPath(t *testing.T) {
	tmpRestore := t.TempDir()

	err := restore("/nonexistent/backup", tmpRestore)
	if err == nil {
		t.Error("expected error with invalid backup path, got nil")
	}
}

func TestRestore_CorruptedChunk(t *testing.T) {
	tmpBackup := t.TempDir()
	tmpRestore := t.TempDir()

	// Create a valid chunk
	chunk := Chunk{
		Entries: []*FileEntry{
			{Path: "file.txt", Mode: 0644, Content: []byte("content")},
		},
	}
	if err := writeChunk(tmpBackup, 1000, 0, chunk); err != nil {
		t.Fatal(err)
	}

	// Create a corrupted chunk
	corruptFile := filepath.Join(tmpBackup, "chunk_2000_000.dat")
	if err := os.WriteFile(corruptFile, []byte("corrupted data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Restore should skip corrupted chunk and restore valid one
	err := restore(tmpBackup, tmpRestore)
	if err != nil {
		t.Fatalf("restore() error = %v", err)
	}

	// Valid file should still be restored
	if _, err := os.Stat(filepath.Join(tmpRestore, "file.txt")); os.IsNotExist(err) {
		t.Error("valid file should be restored despite corrupted chunk")
	}
}

func TestRestore_MultipleChunksSameTimestamp(t *testing.T) {
	tmpBackup := t.TempDir()
	tmpRestore := t.TempDir()

	// Multiple chunks from same backup run
	chunk1 := Chunk{
		Entries: []*FileEntry{
			{Path: "file1.txt", Mode: 0644, Content: []byte("content1")},
		},
	}
	chunk2 := Chunk{
		Entries: []*FileEntry{
			{Path: "file2.txt", Mode: 0644, Content: []byte("content2")},
		},
	}

	timestamp := int64(1000)
	if err := writeChunk(tmpBackup, timestamp, 0, chunk1); err != nil {
		t.Fatal(err)
	}
	if err := writeChunk(tmpBackup, timestamp, 1, chunk2); err != nil {
		t.Fatal(err)
	}

	err := restore(tmpBackup, tmpRestore)
	if err != nil {
		t.Fatalf("restore() error = %v", err)
	}

	// Both files should be restored
	if _, err := os.Stat(filepath.Join(tmpRestore, "file1.txt")); os.IsNotExist(err) {
		t.Error("file1.txt should exist")
	}
	if _, err := os.Stat(filepath.Join(tmpRestore, "file2.txt")); os.IsNotExist(err) {
		t.Error("file2.txt should exist")
	}
}
