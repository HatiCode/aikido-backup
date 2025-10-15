package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type FileEntry struct {
	Path    string
	Mode    os.FileMode
	ModTime time.Time
	Size    int64
	Content []byte
	Deleted bool
}

type Chunk struct {
	Entries []*FileEntry
}

const chunkSize = 5 * 1024 * 1024

func createBackup(backupPath string, entries []*FileEntry) error {
	timestamp := time.Now().Unix()
	chunkNum := 0
	var currentChunk Chunk
	currentSize := 0

	for _, entry := range entries {
		entrySize := len(entry.Content) + 1024

		if currentSize+entrySize > chunkSize && len(currentChunk.Entries) > 0 {
			if err := writeChunk(backupPath, timestamp, chunkNum, currentChunk); err != nil {
				return err
			}
			chunkNum++
			currentChunk = Chunk{}
			currentSize = 0
		}

		currentChunk.Entries = append(currentChunk.Entries, entry)
		currentSize += entrySize
	}

	if len(currentChunk.Entries) > 0 {
		if err := writeChunk(backupPath, timestamp, chunkNum, currentChunk); err != nil {
			return err
		}
	}

	return nil
}

func writeChunk(backupPath string, timestamp int64, num int, chunk Chunk) error {
	filename := filepath.Join(backupPath, fmt.Sprintf("chunk_%d_%03d.dat", timestamp, num))
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return gob.NewEncoder(file).Encode(chunk)
}
