package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashFile(t *testing.T) {
	tests := []struct {
		name     string
		content1 string
		content2 string
		wantSame bool
	}{
		{
			name:     "identical content produces same hash",
			content1: "hello world",
			content2: "hello world",
			wantSame: true,
		},
		{
			name:     "different content produces different hash",
			content1: "hello world",
			content2: "hello world!",
			wantSame: false,
		},
		{
			name:     "empty files produce same hash",
			content1: "",
			content2: "",
			wantSame: true,
		},
		{
			name:     "case sensitive",
			content1: "Hello",
			content2: "hello",
			wantSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			file1 := filepath.Join(tmpDir, "file1.txt")
			file2 := filepath.Join(tmpDir, "file2.txt")

			if err := os.WriteFile(file1, []byte(tt.content1), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file2, []byte(tt.content2), 0644); err != nil {
				t.Fatal(err)
			}

			hash1, err := hashFile(file1)
			if err != nil {
				t.Fatalf("hashFile(file1) error = %v", err)
			}

			hash2, err := hashFile(file2)
			if err != nil {
				t.Fatalf("hashFile(file2) error = %v", err)
			}

			if tt.wantSame && hash1 != hash2 {
				t.Errorf("expected same hash, got %s and %s", hash1, hash2)
			}
			if !tt.wantSame && hash1 == hash2 {
				t.Errorf("expected different hashes, got %s for both", hash1)
			}
		})
	}
}

func TestHashFile_NonExistentFile(t *testing.T) {
	_, err := hashFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestDetectChanges_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	snapshot := make(map[string]string)

	testFile := filepath.Join(tmpDir, "new.txt")
	content := []byte("new file content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	changes, err := detectChanges(tmpDir, snapshot)
	if err != nil {
		t.Fatalf("detectChanges() error = %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	change := changes[0]
	if change.Path != "new.txt" {
		t.Errorf("expected path 'new.txt', got %s", change.Path)
	}
	if change.Deleted {
		t.Error("expected Deleted = false for new file")
	}
	if string(change.Content) != string(content) {
		t.Error("content mismatch")
	}

	if len(snapshot) != 1 {
		t.Errorf("expected snapshot size 1, got %d", len(snapshot))
	}
}

func TestDetectChanges_ModifiedFile(t *testing.T) {
	tmpDir := t.TempDir()
	snapshot := make(map[string]string)

	testFile := filepath.Join(tmpDir, "file.txt")

	if err := os.WriteFile(testFile, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := detectChanges(tmpDir, snapshot)
	if err != nil {
		t.Fatal(err)
	}

	newContent := []byte("modified content")
	if err := os.WriteFile(testFile, newContent, 0644); err != nil {
		t.Fatal(err)
	}

	changes, err := detectChanges(tmpDir, snapshot)
	if err != nil {
		t.Fatalf("detectChanges() error = %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	change := changes[0]
	if change.Path != "file.txt" {
		t.Errorf("expected path 'file.txt', got %s", change.Path)
	}
	if change.Deleted {
		t.Error("expected Deleted = false for modified file")
	}
	if string(change.Content) != string(newContent) {
		t.Error("content should be updated")
	}
}

func TestDetectChanges_DeletedFile(t *testing.T) {
	tmpDir := t.TempDir()
	snapshot := make(map[string]string)

	testFile := filepath.Join(tmpDir, "file.txt")

	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := detectChanges(tmpDir, snapshot)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(testFile); err != nil {
		t.Fatal(err)
	}

	changes, err := detectChanges(tmpDir, snapshot)
	if err != nil {
		t.Fatalf("detectChanges() error = %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	change := changes[0]
	if change.Path != "file.txt" {
		t.Errorf("expected path 'file.txt', got %s", change.Path)
	}
	if !change.Deleted {
		t.Error("expected Deleted = true for deleted file")
	}

	if len(snapshot) != 0 {
		t.Errorf("expected empty snapshot after deletion, got size %d", len(snapshot))
	}
}

func TestDetectChanges_NoChanges(t *testing.T) {
	tmpDir := t.TempDir()
	snapshot := make(map[string]string)

	testFile := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := detectChanges(tmpDir, snapshot)
	if err != nil {
		t.Fatal(err)
	}

	changes, err := detectChanges(tmpDir, snapshot)
	if err != nil {
		t.Fatalf("detectChanges() error = %v", err)
	}

	if len(changes) != 0 {
		t.Errorf("expected no changes, got %d", len(changes))
	}
}

func TestDetectChanges_MultipleChanges(t *testing.T) {
	tmpDir := t.TempDir()
	snapshot := make(map[string]string)

	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	file3 := filepath.Join(tmpDir, "file3.txt")

	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := detectChanges(tmpDir, snapshot)
	if err != nil {
		t.Fatal(err)
	}

	os.WriteFile(file1, []byte("modified1"), 0644)
	os.Remove(file2)
	os.WriteFile(file3, []byte("new"), 0644)

	changes, err := detectChanges(tmpDir, snapshot)
	if err != nil {
		t.Fatalf("detectChanges() error = %v", err)
	}

	if len(changes) != 3 {
		t.Fatalf("expected 3 changes, got %d", len(changes))
	}

	var modified, deleted, new int
	for _, change := range changes {
		switch {
		case change.Path == "file1.txt" && !change.Deleted:
			modified++
		case change.Path == "file2.txt" && change.Deleted:
			deleted++
		case change.Path == "file3.txt" && !change.Deleted:
			new++
		}
	}

	if modified != 1 || deleted != 1 || new != 1 {
		t.Errorf("expected 1 modified, 1 deleted, 1 new; got %d, %d, %d",
			modified, deleted, new)
	}
}

func TestDetectChanges_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	snapshot := make(map[string]string)

	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	file1 := filepath.Join(tmpDir, "root.txt")
	file2 := filepath.Join(subDir, "nested.txt")

	if err := os.WriteFile(file1, []byte("root content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("nested content"), 0644); err != nil {
		t.Fatal(err)
	}

	changes, err := detectChanges(tmpDir, snapshot)
	if err != nil {
		t.Fatalf("detectChanges() error = %v", err)
	}

	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}

	foundNested := false
	for _, change := range changes {
		if change.Path == filepath.Join("subdir", "nested.txt") {
			foundNested = true
		}
	}

	if !foundNested {
		t.Error("nested file not found with correct relative path")
	}
}

func TestDetectChanges_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	snapshot := make(map[string]string)

	changes, err := detectChanges(tmpDir, snapshot)
	if err != nil {
		t.Fatalf("detectChanges() error = %v", err)
	}

	if len(changes) != 0 {
		t.Errorf("expected no changes in empty directory, got %d", len(changes))
	}
}
