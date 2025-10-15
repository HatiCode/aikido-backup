package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
)

func restore(backupPath, restorePath string) error {
	log.Printf("Restoring from %s to %s", backupPath, restorePath)

	if err := os.MkdirAll(restorePath, 0755); err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join(backupPath, "chunk_*.dat"))
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no backup chunks found in %s", backupPath)
	}

	sort.Strings(files)

	fileData := make(map[string]*FileEntry)
	deletedFiles := make(map[string]bool)

	for _, chunkFile := range files {
		chunk, err := readChunk(chunkFile)
		if err != nil {
			log.Printf("Error reading %s: %v", chunkFile, err)
			continue
		}

		for _, entry := range chunk.Entries {
			if entry.Deleted {
				deletedFiles[entry.Path] = true
				delete(fileData, entry.Path)
			} else {
				delete(deletedFiles, entry.Path)
				fileData[entry.Path] = entry
			}
		}
	}

	for _, entry := range fileData {
		targetPath := filepath.Join(restorePath, entry.Path)

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		if err := os.WriteFile(targetPath, entry.Content, entry.Mode); err != nil {
			return err
		}

		if err := os.Chtimes(targetPath, entry.ModTime, entry.ModTime); err != nil {
			log.Printf("Warning: could not restore times for %s", entry.Path)
		}
	}

	log.Printf("Restored %d files", len(fileData))
	return nil
}

func readChunk(filename string) (Chunk, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Chunk{}, err
	}
	defer file.Close()

	var chunk Chunk
	err = gob.NewDecoder(file).Decode(&chunk)
	return chunk, err
}
