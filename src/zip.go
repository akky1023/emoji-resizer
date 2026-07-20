package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type zipItem struct {
	absPath  string
	fileName string
}

type dirZipData struct {
	suffixName string
	items      []zipItem
	entries    []MisskeyEmojiEntry
}

type MisskeyEmojiInfo struct {
	Name        string   `json:"name"`
	Category    string   `json:"category,omitempty"`
	Host        *string  `json:"host,omitempty"`
	Aliases     []string `json:"aliases"`
	License     string   `json:"license,omitempty"`
	IsSensitive *bool    `json:"isSensitive,omitempty"`
	LocalOnly   *bool    `json:"localOnly,omitempty"`
}

type MisskeyEmojiEntry struct {
	FileName   string           `json:"fileName"`
	Downloaded bool             `json:"downloaded"`
	Emoji      MisskeyEmojiInfo `json:"emoji"`
}

type MisskeyMeta struct {
	MetaVersion int                  `json:"metaVersion"`
	Host        *string              `json:"host,omitempty"`
	ExportedAt  string               `json:"exportedAt"`
	Emojis      []MisskeyEmojiEntry `json:"emojis"`
}

func addUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}

func createEmojiZip(zipPath string, items []zipItem, entries []MisskeyEmojiEntry) (err error) {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer func() {
		if closeErr := zipFile.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("failed to close zip file: %w", closeErr)
		}
	}()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	// 1. Create meta.json
	metaFileWriter, err := archive.Create("meta.json")
	if err != nil {
		return fmt.Errorf("failed to create meta.json in zip: %w", err)
	}

	meta := MisskeyMeta{
		MetaVersion: 2,
		ExportedAt:  time.Now().UTC().Format(time.RFC3339),
		Emojis:      entries,
	}
	enc := json.NewEncoder(metaFileWriter)
	enc.SetIndent("", "  ")
	if err := enc.Encode(meta); err != nil {
		return fmt.Errorf("failed to encode and write meta.json: %w", err)
	}

	// 2. Add each image
	for _, item := range items {
		if err := addFileToZip(archive, item); err != nil {
			return err
		}
	}

	if closeErr := archive.Close(); closeErr != nil {
		return fmt.Errorf("failed to close zip archive: %w", closeErr)
	}

	return nil
}

func addFileToZip(archive *zip.Writer, item zipItem) error {
	imgFile, err := os.Open(item.absPath)
	if err != nil {
		return fmt.Errorf("failed to open processed image %s: %w", item.absPath, err)
	}
	defer imgFile.Close()

	imgFileWriter, err := archive.Create(item.fileName)
	if err != nil {
		return fmt.Errorf("failed to create file %s in zip: %w", item.fileName, err)
	}

	if _, err := io.Copy(imgFileWriter, imgFile); err != nil {
		return fmt.Errorf("failed to copy file %s to zip: %w", item.fileName, err)
	}

	return nil
}

// createZipArchives generates local and all-in-one ZIP archives.
func createZipArchives(
	dirZips map[string]*dirZipData,
	allZipItems []zipItem,
	allEmojiEntries []MisskeyEmojiEntry,
	recursive bool,
	topLevelOutDir string,
	category string,
) error {
	categoryPrefix := "emoji"
	if category != "" {
		categoryPrefix = sanitizeFileName(category)
		if categoryPrefix == "" {
			categoryPrefix = "emoji"
		}
	}

	var targetDirs []string
	for k := range dirZips {
		targetDirs = append(targetDirs, k)
	}
	sort.Strings(targetDirs)

	for _, targetDir := range targetDirs {
		data := dirZips[targetDir]
		var zipPath string
		if recursive {
			if err := os.MkdirAll(topLevelOutDir, 0755); err != nil {
				return fmt.Errorf("failed to create top-level output directory: %w", err)
			}
			zipPath = filepath.Join(topLevelOutDir, categoryPrefix+"_"+data.suffixName+".zip")
		} else {
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
			}
			zipPath = filepath.Join(targetDir, categoryPrefix+".zip")
		}

		displayZipPath := filepath.Clean(zipPath)
		fmt.Printf("Creating ZIP archive at %s ... ", displayZipPath)
		if err := createEmojiZip(zipPath, data.items, data.entries); err != nil {
			fmt.Println("Failed")
			return err
		}
		fmt.Println("Success")
	}

	if recursive {
		if err := os.MkdirAll(topLevelOutDir, 0755); err != nil {
			return fmt.Errorf("failed to create top-level output directory: %w", err)
		}
		zipPath := filepath.Join(topLevelOutDir, categoryPrefix+"_all.zip")
		displayZipPath := filepath.Clean(zipPath)
		fmt.Printf("Creating ZIP archive at %s ... ", displayZipPath)
		if err := createEmojiZip(zipPath, allZipItems, allEmojiEntries); err != nil {
			fmt.Println("Failed")
			return err
		}
		fmt.Println("Success")
	}

	return nil
}

