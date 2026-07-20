package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var version = "devel"



func main() {
	var (
		size            int
		outDir          string
		suffix          string
		namePrefix      string
		nameSuffix      string
		configPath      string
		recursive       bool
		noResize        bool
		noResizeIfSmall bool
		rect            bool
		showVersion     bool
		zipMode         bool
		autoRect        AutoRectValue
		skip            bool
		checkMode       bool
	)

	flag.IntVar(&size, "size", 128, "target resize square size in pixels")
	flag.StringVar(&outDir, "out", "", "custom output directory path (default: 'output' directory inside the source file's directory)")
	flag.StringVar(&suffix, "suffix", "", "suffix to append to the output filename (e.g. '_resized')")
	flag.StringVar(&namePrefix, "name-prefix", "", "prefix to prepend to the emoji name")
	flag.StringVar(&nameSuffix, "name-suffix", "", "suffix to append to the emoji name")
	flag.StringVar(&configPath, "config", "", "path to config file (default: './config.json' if exists)")
	flag.BoolVar(&recursive, "r", false, "recursively scan directories")
	flag.BoolVar(&noResize, "no-resize", false, "skip final resizing and keep the original square dimensions")
	flag.BoolVar(&noResizeIfSmall, "no-resize-if-small", false, "skip resizing if the image is already smaller than the target size")
	flag.BoolVar(&rect, "rect", false, "resize rectangle keeping aspect ratio, short side matches target size (no padding)")
	flag.BoolVar(&showVersion, "version", false, "show version information and exit")
	flag.BoolVar(&zipMode, "zip", false, "pack processed images into a Misskey-compatible emoji ZIP file")
	flag.Var(&autoRect, "auto-rect", "automatically use rect mode if aspect ratio exceeds threshold (defaults to golden ratio ~1.618)")
	flag.BoolVar(&skip, "skip", false, "skip resizing if the destination file already exists")
	flag.BoolVar(&checkMode, "check", false, "check for duplicate emoji names after conversion")
	// Pre-process os.Args to handle optional argument for -config
	var newArgs []string
	for i := 0; i < len(os.Args); i++ {
		newArgs = append(newArgs, os.Args[i])
		if os.Args[i] == "-config" || os.Args[i] == "--config" {
			// Check if the next argument is missing or is another flag
			nextIsVal := false
			if i+1 < len(os.Args) {
				if !strings.HasPrefix(os.Args[i+1], "-") {
					nextIsVal = true
				}
			}
			if !nextIsVal {
				// No argument specified for -config. Determine default value.
				defaultPath := "config.json"
				if info, err := os.Stat("config"); err == nil && !info.IsDir() {
					defaultPath = "config"
				} else if info, err := os.Stat("config.json"); err == nil && !info.IsDir() {
					defaultPath = "config.json"
				}
				newArgs = append(newArgs, defaultPath)
			}
		}
	}
	os.Args = newArgs

	flag.Parse()

	if showVersion {
		fmt.Printf("emoji-resizer %s\n", version)
		return
	}

	seenFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		seenFlags[f.Name] = true
	})

	var cfgCategory string
	var cfgLicense string

	if err := parseAndApplyConfig(configPath, seenFlags, &size, &outDir, &suffix, &namePrefix, &nameSuffix, &recursive, &noResize, &rect, &zipMode, &skip, &autoRect, &cfgCategory, &cfgLicense, &noResizeIfSmall); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if size < 8 || size > 8192 {
		fmt.Fprintf(os.Stderr, "Error: invalid size %d (must be between 8 and 8192 px)\n", size)
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) == 0 {
		// Default to current directory if no files/folders specified
		args = []string{"."}
	}

	// Resolve absolute path for custom output directory if specified
	var absOutDir string
	if outDir != "" {
		var err error
		absOutDir, err = filepath.Abs(outDir)
		if err != nil {
			fmt.Printf("Warning: failed to resolve absolute path for output directory: %v\n", err)
		}
	}

	// Ask for category and license if zipMode is active
	reader := bufio.NewReader(os.Stdin)
	var category string
	var license string
	if zipMode && !checkMode {
		if cfgCategory != "" {
			category = cfgCategory
		} else {
			fmt.Print("emoji.category を入力してください (スキップするにはEnter): ")
			catInput, err := reader.ReadString('\n')
			if err == nil {
				category = strings.TrimSpace(catInput)
			}
		}

		if cfgLicense != "" {
			license = cfgLicense
		} else {
			fmt.Print("emoji.license を入力してください (スキップするにはEnter): ")
			licInput, err := reader.ReadString('\n')
			if err == nil {
				license = strings.TrimSpace(licInput)
			}
		}
	}

	// Collect files to process
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

	if len(filesToProcess) == 0 {
		if checkMode {
			fmt.Println("OK")
			os.Exit(0)
		}
		fmt.Println("No supported image files found to process.")
		return
	}

	if checkMode {
		nameToPaths := make(map[string][]string)
		var candidateNamesOrdered []string

		for _, filePath := range filesToProcess {
			displayPath := filepath.Clean(filePath)

			// Collect candidate names for this file
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
			normalName := namePrefix + base + nameSuffix
			addCandidate(normalName)

			// 2. ZIP main name
			zipBase, _, _, _, _, _, _ := computeEmojiName(filePath, true, namePrefix, nameSuffix, reader)
			addCandidate(zipBase)

			// Map each unique candidate name to this file path
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
				// Deduplicate duplicate groups by their contents to prevent redundant outputs
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

	if noResize {
		var modeStr string
		if rect {
			modeStr = "no-resize: rect mode"
		} else if autoRect.Active {
			if autoRect.Ratio > 1.0 {
				modeStr = fmt.Sprintf("no-resize: auto-rect mode threshold %g", autoRect.Ratio)
			} else {
				modeStr = "no-resize: auto-rect mode threshold golden ratio"
			}
		} else {
			modeStr = "no-resize: padding only"
		}
		fmt.Printf("Found %d image files. Starting processing (%s)...\n", len(filesToProcess), modeStr)
	} else {
		if rect {
			var extra string
			if noResizeIfSmall {
				extra = ", no-resize-if-small"
			}
			fmt.Printf("Found %d image files. Starting processing (rect mode, target short side: %d px%s)...\n", len(filesToProcess), size, extra)
		} else if autoRect.Active {
			var thStr string
			if autoRect.Ratio > 1.0 {
				thStr = fmt.Sprintf("%g", autoRect.Ratio)
			} else {
				thStr = "golden ratio"
			}
			var extra string
			if noResizeIfSmall {
				extra = ", no-resize-if-small"
			}
			fmt.Printf("Found %d image files. Starting processing (auto-rect mode threshold %s, target size: %d px%s)...\n", len(filesToProcess), thStr, size, extra)
		} else {
			var extra string
			if noResizeIfSmall {
				extra = ", no-resize-if-small"
			}
			fmt.Printf("Found %d image files. Starting processing (target size: %dx%d px%s)...\n", len(filesToProcess), size, size, extra)
		}
	}

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

	dirZips := make(map[string]*dirZipData)
	var allZipItems []zipItem
	var allEmojiEntries []MisskeyEmojiEntry

	successCount := 0
	failureCount := 0
	for _, filePath := range filesToProcess {
		// Use backslashes on Windows for clean output log paths
		displayPath := filepath.Clean(filePath)
		fmt.Printf("Processing %s ... ", displayPath)

		customBase, name, hiragana, katakana, hepburn, hasPronunciation, rawAliases := computeEmojiName(filePath, zipMode, namePrefix, nameSuffix, reader)

		destPath, skipped, err := processImage(filePath, outDir, size, suffix, noResize, rect, customBase, autoRect.Active, autoRect.Ratio, skip, noResizeIfSmall)
		if err != nil {
			fmt.Printf("Failed: %v\n", err)
			failureCount++
		} else {
			if skipped {
				fmt.Println("Skipped")
			} else {
				fmt.Println("Success")
			}
			successCount++

			if zipMode {
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
						Name:    namePrefix + name + nameSuffix,
						Aliases: aliases,
					},
				}
				if category != "" {
					entry.Emoji.Category = category
				}
				if license != "" {
					entry.Emoji.License = license
				}
				data.entries = append(data.entries, entry)
				allEmojiEntries = append(allEmojiEntries, entry)
			}
		}
	}

	if zipMode && successCount > 0 {
		categoryPrefix := "emoji"
		if category != "" {
			categoryPrefix = sanitizeFileName(category)
			if categoryPrefix == "" {
				categoryPrefix = "emoji"
			}
		}

		// 1. Create local emojis.zip for each output directory
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
					fmt.Printf("Failed to create top-level output directory: %v\n", err)
					os.Exit(1)
				}
				zipPath = filepath.Join(topLevelOutDir, categoryPrefix+"_"+data.suffixName+".zip")
			} else {
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					fmt.Printf("Failed to create directory %s: %v\n", targetDir, err)
					os.Exit(1)
				}
				zipPath = filepath.Join(targetDir, categoryPrefix+".zip")
			}
			displayZipPath := filepath.Clean(zipPath)
			fmt.Printf("Creating ZIP archive at %s ... ", displayZipPath)
			err := createEmojiZip(zipPath, data.items, data.entries)
			if err != nil {
				fmt.Printf("Failed: %v\n", err)
				os.Exit(1)
			} else {
				fmt.Println("Success")
			}
		}

		// 2. If recursive, also create a top-level allemoji.zip containing all emojis
		if recursive {
			if err := os.MkdirAll(topLevelOutDir, 0755); err != nil {
				fmt.Printf("Failed to create top-level output directory: %v\n", err)
				os.Exit(1)
			}
			zipPath := filepath.Join(topLevelOutDir, categoryPrefix+"_all.zip")
			displayZipPath := filepath.Clean(zipPath)
			fmt.Printf("Creating ZIP archive at %s ... ", displayZipPath)
			err := createEmojiZip(zipPath, allZipItems, allEmojiEntries)
			if err != nil {
				fmt.Printf("Failed: %v\n", err)
				os.Exit(1)
			} else {
				fmt.Println("Success")
			}
		}
	}

	fmt.Printf("Finished. Successfully processed %d/%d files.\n", successCount, len(filesToProcess))
	if failureCount > 0 {
		os.Exit(1)
	}
}


