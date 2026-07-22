package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// collectFilesToProcess scans given arguments and returns a slice of supported image file paths.
func collectFilesToProcess(args []string, recursive bool, outDir, absOutDir string, checkMode bool) []string {
	var filesToProcess []string

	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			if checkMode {
				fmt.Fprintf(os.Stderr, "Error accessing path %s: %v\n", arg, err)
			} else {
				fmt.Printf("Error accessing path %s: %v\n", arg, err)
			}
			continue
		}

		if info.IsDir() {
			scanned, err := scanDirectory(arg, recursive, outDir, absOutDir)
			if err != nil {
				if checkMode {
					fmt.Fprintf(os.Stderr, "Error scanning directory %s: %v\n", arg, err)
				} else {
					fmt.Printf("Error scanning directory %s: %v\n", arg, err)
				}
				continue
			}
			filesToProcess = append(filesToProcess, scanned...)
		} else {
			if isSupportedExtension(arg) {
				filesToProcess = append(filesToProcess, arg)
			} else {
				if checkMode {
					fmt.Fprintf(os.Stderr, "Skipping unsupported file format: %s\n", arg)
				} else {
					fmt.Printf("Skipping unsupported file format: %s\n", arg)
				}
			}
		}
	}

	return filesToProcess
}

// executeCheckMode checks for duplicate emoji names and exits.
func executeCheckMode(filesToProcess []string, zipMode bool, namePrefix, nameSuffix string, reader *bufio.Reader, filenameOption bool) {
	nameToPaths := make(map[string][]string)
	var candidateNamesOrdered []string

	for _, filePath := range filesToProcess {
		displayPath := filepath.Clean(filePath)

		seenCandidates := make(map[string]bool)
		var uniqueCandidates []string

		addCandidate := func(name string) {
			if name != "" && !seenCandidates[name] {
				seenCandidates[name] = true
				uniqueCandidates = append(uniqueCandidates, name)
			}
		}

		// 1. Normal name (prefixed and suffixed)
		ext := filepath.Ext(filePath)
		base := strings.TrimSuffix(filepath.Base(filePath), ext)
		if filenameOption {
			fnOpt := parseFilenameOption(filePath)
			if fnOpt.CleanRawBase != "" {
				base = fnOpt.CleanRawBase
			}
		}
		normalName := namePrefix + base + nameSuffix
		addCandidate(normalName)

		// 2. ZIP main name
		zipBase, _, _, _, _, _, _ := computeEmojiName(filePath, true, namePrefix, nameSuffix, reader, filenameOption)
		addCandidate(zipBase)

		for _, candidate := range uniqueCandidates {
			if len(nameToPaths[candidate]) == 0 {
				candidateNamesOrdered = append(candidateNamesOrdered, candidate)
			}
			nameToPaths[candidate] = append(nameToPaths[candidate], displayPath)
		}
	}

	printedGroups := make(map[string]bool)
	var duplicateGroups [][]string

	for _, name := range candidateNamesOrdered {
		paths := nameToPaths[name]
		if len(paths) > 1 {
			sortedPaths := make([]string, len(paths))
			copy(sortedPaths, paths)
			sort.Strings(sortedPaths)

			key := strings.Join(sortedPaths, "|")
			if !printedGroups[key] {
				printedGroups[key] = true
				duplicateGroups = append(duplicateGroups, paths)
			}
		}
	}

	if len(duplicateGroups) == 0 {
		fmt.Println("OK")
		os.Exit(0)
	} else {
		for i, group := range duplicateGroups {
			if i > 0 {
				fmt.Println("===")
			}
			for _, path := range group {
				fmt.Println(path)
			}
		}
		os.Exit(1)
	}
}

// printStartMessage prints the starting information log before processing images.
func printStartMessage(filesCount int, opts *appOptions) {
	var extra string
	if opts.noResizeIfSmall {
		extra = ", no-resize-if-small"
	}

	if opts.noResize {
		var modeStr string
		if opts.rect {
			modeStr = "no-resize: rect mode"
		} else if opts.autoRect.Active {
			if opts.autoRect.Ratio > 1.0 {
				modeStr = fmt.Sprintf("no-resize: auto-rect mode threshold %g", opts.autoRect.Ratio)
			} else {
				modeStr = "no-resize: auto-rect mode threshold golden ratio"
			}
		} else {
			modeStr = "no-resize: padding only"
		}
		fmt.Printf("Found %d image files. Starting processing (%s)...\n", filesCount, modeStr)
	} else {
		if opts.rect {
			fmt.Printf("Found %d image files. Starting processing (rect mode, target short side: %d px%s)...\n", filesCount, opts.size, extra)
		} else if opts.autoRect.Active {
			var thStr string
			if opts.autoRect.Ratio > 1.0 {
				thStr = fmt.Sprintf("%g", opts.autoRect.Ratio)
			} else {
				thStr = "golden ratio"
			}
			fmt.Printf("Found %d image files. Starting processing (auto-rect mode threshold %s, target size: %d px%s)...\n", filesCount, thStr, opts.size, extra)
		} else {
			fmt.Printf("Found %d image files. Starting processing (target size: %dx%d px%s)...\n", filesCount, opts.size, opts.size, extra)
		}
	}
}

// resolveDirectoryPaths calculates top level input and output directory paths.
func resolveDirectoryPaths(args []string, outDir, absOutDir string) (string, string) {
	var topLevelInDir string
	if len(args) > 0 {
		topLevelInDir = args[0]
	} else {
		topLevelInDir = "."
	}

	absTopLevelInDir, err := filepath.Abs(topLevelInDir)
	if err != nil {
		fmt.Printf("Warning: failed to resolve absolute path for top-level input directory: %v\n", err)
		absTopLevelInDir = topLevelInDir
	}

	var topLevelOutDir string
	if outDir != "" {
		topLevelOutDir = absOutDir
	} else if len(args) > 0 {
		arg0 := args[0]
		info, err := os.Stat(arg0)
		if err == nil {
			if info.IsDir() {
				topLevelOutDir = filepath.Join(arg0, "output")
			} else {
				topLevelOutDir = filepath.Join(filepath.Dir(arg0), "output")
			}
		}
	}

	if topLevelOutDir != "" {
		topLevelOutDir = filepath.Clean(topLevelOutDir)
	} else {
		topLevelOutDir = "output"
	}

	return absTopLevelInDir, topLevelOutDir
}

// processBatchImages processes each image and gathers zip data if zipMode is enabled.
func processBatchImages(filesToProcess []string, opts *appOptions, reader *bufio.Reader, absTopLevelInDir string) (int, int, map[string]*dirZipData, []zipItem, []MisskeyEmojiEntry) {
	dirZips := make(map[string]*dirZipData)
	var allZipItems []zipItem
	var allEmojiEntries []MisskeyEmojiEntry

	successCount := 0
	failureCount := 0

	for _, filePath := range filesToProcess {
		displayPath := filepath.Clean(filePath)
		fmt.Printf("Processing %s ... ", displayPath)

		customBase, name, hiragana, katakana, hepburn, hasPronunciation, rawAliases := computeEmojiName(
			filePath, opts.zipMode, opts.namePrefix, opts.nameSuffix, reader, opts.filenameOption,
		)

		actualRect := opts.rect
		actualAutoRectActive := opts.autoRect.Active

		if opts.filenameOption {
			fnOpt := parseFilenameOption(filePath)
			if fnOpt.HasR {
				actualRect = true
				actualAutoRectActive = false
			} else if fnOpt.HasS {
				actualRect = false
				actualAutoRectActive = false
			}
		}

		destPath, skipped, err := processImage(
			filePath, opts.outDir, opts.size, opts.suffix, opts.noResize, actualRect,
			customBase, actualAutoRectActive, opts.autoRect.Ratio, opts.skip, opts.noResizeIfSmall,
		)

		if err != nil {
			fmt.Printf("Failed: %v\n", err)
			failureCount++
			continue
		}

		if skipped {
			fmt.Println("Skipped")
		} else {
			fmt.Println("Success")
		}
		successCount++

		if opts.zipMode {
			outFileName := filepath.Base(destPath)
			targetDir := filepath.Dir(destPath)

			data, ok := dirZips[targetDir]
			if !ok {
				var suffixName string
				absSrcDir, err := filepath.Abs(filepath.Dir(filePath))
				if err != nil {
					absSrcDir = filepath.Dir(filePath)
				}
				rel, err := filepath.Rel(absTopLevelInDir, absSrcDir)
				if err != nil || rel == "." || rel == "" {
					suffixName = filepath.Base(absTopLevelInDir)
				} else {
					relClean := filepath.Clean(rel)
					relClean = strings.ReplaceAll(relClean, "\\", "_")
					relClean = strings.ReplaceAll(relClean, "/", "_")
					suffixName = relClean
				}
				suffixName = sanitizeFileName(suffixName)

				data = &dirZipData{
					suffixName: suffixName,
				}
				dirZips[targetDir] = data
			}

			item := zipItem{
				absPath:  destPath,
				fileName: outFileName,
			}
			data.items = append(data.items, item)
			allZipItems = append(allZipItems, item)

			aliases := []string{}
			if hasPronunciation {
				aliases = addUnique(aliases, hiragana)
				aliases = addUnique(aliases, katakana)
				if name != hepburn {
					aliases = addUnique(aliases, hepburn)
				}
			}
			for _, extra := range rawAliases {
				for _, exp := range expandAlias(extra) {
					aliases = addUnique(aliases, exp)
				}
			}

			entry := MisskeyEmojiEntry{
				FileName:   outFileName,
				Downloaded: true,
				Emoji: MisskeyEmojiInfo{
					Name:    opts.namePrefix + name + opts.nameSuffix,
					Aliases: aliases,
				},
			}
			if opts.category != "" {
				entry.Emoji.Category = opts.category
			}
			if opts.license != "" {
				entry.Emoji.License = opts.license
			}
			data.entries = append(data.entries, entry)
			allEmojiEntries = append(allEmojiEntries, entry)
		}
	}

	return successCount, failureCount, dirZips, allZipItems, allEmojiEntries
}
