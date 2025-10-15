package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"maps"
	"os"
	"path/filepath"
	"time"
)

func watch(watchPath string, backupPath string, refresh int) error {
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return err
	}
	log.Printf("Watching %s, backing up to %s every %d seconds\n",
		watchPath, backupPath, refresh)

	snapshot := make(map[string]string)

	for {
		changes, err := detectChanges(watchPath, snapshot)
		if err != nil {
			log.Printf("Error detecting changes: %v", err)
		}

		if len(changes) > 0 {
			log.Printf("Detected %d changes, creating backup...", len(changes))
			if err := createBackup(backupPath, changes); err != nil {
				log.Printf("Backup error: %v", err)
			} else {
				log.Println("Backup completed")
			}
		}

		time.Sleep(time.Duration(refresh) * time.Second)
	}

}

func detectChanges(watchPath string, snapshot map[string]string) ([]*FileEntry, error) {
	current := make(map[string]string)
	var changes []*FileEntry

	err := filepath.WalkDir(watchPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(watchPath, path)
		if err != nil {
			return err
		}

		hash, err := hashFile(path)
		if err != nil {
			return err
		}

		current[relPath] = hash

		if oldHash, exists := snapshot[relPath]; !exists || oldHash != hash {
			info, err := d.Info()
			if err != nil {
				return err
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			changes = append(changes, &FileEntry{
				Path:    relPath,
				Mode:    info.Mode(),
				ModTime: info.ModTime(),
				Size:    info.Size(),
				Content: content,
				Deleted: false,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	for oldPath := range snapshot {
		if _, exists := current[oldPath]; !exists {
			changes = append(changes, &FileEntry{
				Path:    oldPath,
				Deleted: true,
			})
		}
	}

	maps.Copy(snapshot, current)
	for oldPath := range snapshot {
		if _, exists := current[oldPath]; !exists {
			delete(snapshot, oldPath)
		}
	}

	return changes, nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
