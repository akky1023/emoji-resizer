package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var version = "devel"

type AutoRectValue struct {
	Active bool
	Ratio  float64
}

func (a *AutoRectValue) String() string {
	if !a.Active {
		return "false"
	}
	if a.Ratio == 0 {
		return "true"
	}
	return fmt.Sprintf("%g", a.Ratio)
}

func (a *AutoRectValue) Set(s string) error {
	a.Active = true
	if s == "" || s == "true" {
		a.Ratio = 0
		return nil
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("invalid auto-rect ratio: %w", err)
	}
	if val <= 1.0 {
		return fmt.Errorf("auto-rect ratio must be greater than 1.0")
	}
	a.Ratio = val
	return nil
}

func (a *AutoRectValue) IsBoolFlag() bool {
	return true
}

func main() {
	var (
		size        int
		outDir      string
		suffix      string
		recursive   bool
		noResize    bool
		rect        bool
		showVersion bool
		zipMode     bool
		autoRect    AutoRectValue
		skip        bool
	)

	flag.IntVar(&size, "size", 128, "target resize square size in pixels")
	flag.StringVar(&outDir, "out", "", "custom output directory path (default: 'output' directory inside the source file's directory)")
	flag.StringVar(&suffix, "suffix", "", "suffix to append to the output filename (e.g. '_resized')")
	flag.BoolVar(&recursive, "r", false, "recursively scan directories")
	flag.BoolVar(&noResize, "no-resize", false, "skip final resizing and keep the original square dimensions")
	flag.BoolVar(&rect, "rect", false, "resize rectangle keeping aspect ratio, short side matches target size (no padding)")
	flag.BoolVar(&showVersion, "version", false, "show version information and exit")
	flag.BoolVar(&zipMode, "zip", false, "pack processed images into a Misskey-compatible emoji ZIP file")
	flag.Var(&autoRect, "auto-rect", "automatically use rect mode if aspect ratio exceeds threshold (defaults to golden ratio ~1.618)")
	flag.BoolVar(&skip, "skip", false, "skip resizing if the destination file already exists")
	flag.Parse()

	if showVersion {
		fmt.Printf("emoji-resizer %s\n", version)
		return
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
	var category string
	var license string
	if zipMode {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("emoji.category を入力してください (スキップするにはEnter): ")
		catInput, _ := reader.ReadString('\n')
		category = strings.TrimSpace(catInput)

		fmt.Print("emoji.license を入力してください (スキップするにはEnter): ")
		licInput, _ := reader.ReadString('\n')
		license = strings.TrimSpace(licInput)
	}

	// Collect files to process
	var filesToProcess []string
	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			fmt.Printf("Error accessing path %s: %v\n", arg, err)
			continue
		}

		if info.IsDir() {
			scanned, err := scanDirectory(arg, recursive, outDir, absOutDir)
			if err != nil {
				fmt.Printf("Error scanning directory %s: %v\n", arg, err)
				continue
			}
			filesToProcess = append(filesToProcess, scanned...)
		} else {
			if isSupportedExtension(arg) {
				filesToProcess = append(filesToProcess, arg)
			} else {
				fmt.Printf("Skipping unsupported file format: %s\n", arg)
			}
		}
	}

	if len(filesToProcess) == 0 {
		fmt.Println("No supported image files found to process.")
		return
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
			fmt.Printf("Found %d image files. Starting processing (rect mode, target short side: %d px)...\n", len(filesToProcess), size)
		} else if autoRect.Active {
			var thStr string
			if autoRect.Ratio > 1.0 {
				thStr = fmt.Sprintf("%g", autoRect.Ratio)
			} else {
				thStr = "golden ratio"
			}
			fmt.Printf("Found %d image files. Starting processing (auto-rect mode threshold %s, target size: %d px)...\n", len(filesToProcess), thStr, size)
		} else {
			fmt.Printf("Found %d image files. Starting processing (target size: %dx%d px)...\n", len(filesToProcess), size, size)
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

		var customBase string
		var name, hiragana, katakana, hepburn string
		var hasPronunciation bool

		if zipMode {
			ext := filepath.Ext(filePath)
			base := strings.TrimSuffix(filepath.Base(filePath), ext)

			if isPureHiraganaOrSafe(base) {
				hiragana = base
				katakana = hiraganaToKatakana(hiragana)
				var hepburnRaw string
				name, hepburnRaw = hiraganaToRomaji(hiragana)
				name = strings.ToLower(name)
				hepburn = strings.ToLower(hepburnRaw)
				hasPronunciation = true
			} else if containsJapanese(base) {
				reader := bufio.NewReader(os.Stdin)
				fmt.Printf("ファイル名 '%s' のひらがな表記を入力してください (英語などの場合はそのままEnter): ", base)
				input, _ := reader.ReadString('\n')
				hiragana = strings.TrimSpace(input)
				if hiragana != "" {
					katakana = hiraganaToKatakana(hiragana)
					var hepburnRaw string
					name, hepburnRaw = hiraganaToRomaji(hiragana)
					name = strings.ToLower(name)
					hepburn = strings.ToLower(hepburnRaw)
					hasPronunciation = true
				} else {
					name = strings.ToLower(base)
				}
			} else {
				name = strings.ToLower(base)
			}
			customBase = name
		}

		destPath, skipped, err := processImage(filePath, outDir, size, suffix, noResize, rect, customBase, autoRect.Active, autoRect.Ratio, skip)
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

				entry := MisskeyEmojiEntry{
					FileName:   outFileName,
					Downloaded: true,
					Emoji: MisskeyEmojiInfo{
						Name:    name,
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
