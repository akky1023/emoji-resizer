package main

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/image/draw"

	// pure Go WebP library supporting lossless encoding
	"github.com/deepteams/webp"
)

var version = "devel"

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

type zipItem struct {
	absPath  string
	fileName string
}

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

	var firstOutDir string
	var zipItems []zipItem
	var emojiEntries []MisskeyEmojiEntry

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

		destPath, err := processImage(filePath, outDir, size, suffix, noResize, rect, customBase, autoRect.Active, autoRect.Ratio)
		if err != nil {
			fmt.Printf("Failed: %v\n", err)
			failureCount++
		} else {
			fmt.Println("Success")
			successCount++

			if zipMode {
				if firstOutDir == "" {
					firstOutDir = filepath.Dir(destPath)
				}
				outFileName := filepath.Base(destPath)
				zipItems = append(zipItems, zipItem{
					absPath:  destPath,
					fileName: outFileName,
				})

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
				emojiEntries = append(emojiEntries, entry)
			}
		}
	}

	if zipMode && successCount > 0 {
		var zipPath string
		if firstOutDir != "" {
			zipPath = filepath.Join(firstOutDir, "emojis.zip")
		} else {
			zipPath = "emojis.zip"
		}
		fmt.Printf("Creating ZIP archive at %s ... ", zipPath)
		err := createEmojiZip(zipPath, zipItems, emojiEntries)
		if err != nil {
			fmt.Printf("Failed: %v\n", err)
			os.Exit(1)
		} else {
			fmt.Println("Success")
		}
	}

	fmt.Printf("Finished. Successfully processed %d/%d files.\n", successCount, len(filesToProcess))
	if failureCount > 0 {
		os.Exit(1)
	}
}

func isSupportedExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png", ".webp", ".gif", ".jpg", ".jpeg":
		return true
	}
	return false
}

func scanDirectory(dirPath string, recursive bool, outDir string, absOutDir string) ([]string, error) {
	var files []string

	if recursive {
		err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				// 1. Avoid processing default output directories
				if info.Name() == "output" {
					return filepath.SkipDir
				}
				// 2. Avoid processing custom output directory (exact path match or sub-path match)
				if absOutDir != "" {
					absPath, err := filepath.Abs(path)
					if err == nil {
						if absPath == absOutDir {
							return filepath.SkipDir
						}
						// Check if absPath is a subdirectory of absOutDir
						rel, err := filepath.Rel(absOutDir, absPath)
						if err == nil && !strings.HasPrefix(rel, "..") {
							return filepath.SkipDir
						}
					}
				}
			}
			if !info.IsDir() && isSupportedExtension(path) {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				path := filepath.Join(dirPath, entry.Name())
				if isSupportedExtension(path) {
					files = append(files, path)
				}
			}
		}
	}

	return files, nil
}

func processImage(srcPath string, outDir string, targetSize int, suffix string, noResize bool, rect bool, customBase string, autoRectActive bool, autoRectRatio float64) (string, error) {
	// 1. Open the source file
	file, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 2. Decode the image
	var img image.Image
	var format string
	ext := filepath.Ext(srcPath)
	extLower := strings.ToLower(ext)

	if extLower == ".gif" {
		// Scan for multi-frame animated GIFs to warn the user
		g, errGif := gif.DecodeAll(file)
		if errGif == nil && len(g.Image) > 1 {
			fmt.Printf("(Warning: animated GIF detected, only the first frame will be processed) ")
		}
		// Reset file pointer to read the first frame via the standard Decode path
		_, errSeek := file.Seek(0, 0)
		if errSeek != nil {
			if errGif == nil && len(g.Image) > 0 {
				img = g.Image[0]
				format = "gif"
			} else {
				return "", fmt.Errorf("failed to seek/decode GIF: %w", errSeek)
			}
		}
	}

	if img == nil {
		img, format, err = image.Decode(file)
		if err != nil {
			return "", fmt.Errorf("failed to decode image: %w", err)
		}
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	actualRect := rect
	if autoRectActive && w > 0 && h > 0 {
		var maxDim, minDim float64
		if w > h {
			maxDim = float64(w)
			minDim = float64(h)
		} else {
			maxDim = float64(h)
			minDim = float64(w)
		}
		ratio := maxDim / minDim
		threshold := 1.618033988749895
		if autoRectRatio > 1.0 {
			threshold = autoRectRatio
		}
		actualRect = ratio > threshold
	}

	var finalImg image.Image
	if actualRect {
		if noResize {
			// Create a transparent RGBA canvas to preserve premultiplied alpha (prevents dark fringes)
			canvas := image.NewRGBA(image.Rect(0, 0, w, h))
			draw.Draw(canvas, image.Rect(0, 0, w, h), img, bounds.Min, draw.Src)
			finalImg = canvas
		} else {
			var rectW, rectH int
			if w < h {
				rectW = targetSize
				rectH = int(float64(h)*float64(targetSize)/float64(w) + 0.5)
			} else {
				rectH = targetSize
				rectW = int(float64(w)*float64(targetSize)/float64(h) + 0.5)
			}
			// Draw the source image onto a transparent RGBA canvas first to preserve premultiplied alpha
			canvas := image.NewRGBA(image.Rect(0, 0, w, h))
			draw.Draw(canvas, image.Rect(0, 0, w, h), img, bounds.Min, draw.Src)

			resized := image.NewRGBA(image.Rect(0, 0, rectW, rectH))
			draw.CatmullRom.Scale(resized, resized.Bounds(), canvas, canvas.Bounds(), draw.Src, nil)
			finalImg = resized
		}
	} else {
		// 3. Make square (padding)
		maxDim := w
		if h > w {
			maxDim = h
		}

		// Create a transparent RGBA canvas to preserve premultiplied alpha (prevents dark fringes)
		canvas := image.NewRGBA(image.Rect(0, 0, maxDim, maxDim))

		// Center coordinates
		offsetX := (maxDim - w) / 2
		offsetY := (maxDim - h) / 2

		// Draw the source image onto the center of the canvas
		draw.Draw(canvas, image.Rect(offsetX, offsetY, offsetX+w, offsetY+h), img, bounds.Min, draw.Src)

		// 4. Resize to TargetSize x TargetSize using CatmullRom resampling if resizing is enabled
		if noResize {
			finalImg = canvas
		} else {
			resized := image.NewRGBA(image.Rect(0, 0, targetSize, targetSize))
			draw.CatmullRom.Scale(resized, resized.Bounds(), canvas, canvas.Bounds(), draw.Src, nil)
			finalImg = resized
		}
	}

	// 5. Determine the output path
	base := customBase
	if base == "" {
		base = strings.TrimSuffix(filepath.Base(srcPath), ext)
	}

	// Generate output directory
	finalOutDir := outDir
	if finalOutDir == "" {
		finalOutDir = filepath.Join(filepath.Dir(srcPath), "output")
	}

	if err := os.MkdirAll(finalOutDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	outFileName := base + suffix + ext
	destPath := filepath.Join(finalOutDir, outFileName)

	// Prevent overwriting the original file in any case
	absSrc, err := filepath.Abs(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path for source file: %w", err)
	}
	absDest, err := filepath.Abs(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path for destination file: %w", err)
	}

	if absSrc == absDest {
		// If they are exactly the same, force a suffix to be safe
		outFileName = base + suffix + "_resized" + ext
		destPath = filepath.Join(finalOutDir, outFileName)
		absDest, err = filepath.Abs(destPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve absolute path for safe destination file: %w", err)
		}
	}

	if absSrc == absDest {
		return "", fmt.Errorf("refusing to write: output file path is identical to source file path")
	}

	// Write to a temporary file first, then rename it (atomic swap) to prevent corruption
	tmpPath := destPath + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath) // safe to call if already renamed or closed
	}()

	// 6. Encode the processed image using the appropriate format
	switch extLower {
	case ".png":
		err = png.Encode(tmpFile, finalImg)
	case ".webp":
		err = webp.Encode(tmpFile, finalImg, &webp.EncoderOptions{
			Lossless: true,
			Quality:  100,
		})
	case ".gif":
		err = gif.Encode(tmpFile, finalImg, nil)
	case ".jpg", ".jpeg":
		err = jpeg.Encode(tmpFile, finalImg, &jpeg.Options{Quality: 95})
	default:
		switch format {
		case "png":
			err = png.Encode(tmpFile, finalImg)
		case "webp":
			err = webp.Encode(tmpFile, finalImg, &webp.EncoderOptions{Lossless: true, Quality: 100})
		case "gif":
			err = gif.Encode(tmpFile, finalImg, nil)
		case "jpeg":
			err = jpeg.Encode(tmpFile, finalImg, &jpeg.Options{Quality: 95})
		default:
			err = png.Encode(tmpFile, finalImg)
		}
	}

	if err != nil {
		return "", fmt.Errorf("failed to encode/save image: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temporary file: %w", err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return "", fmt.Errorf("failed to rename temporary file to destination: %w", err)
	}

	return destPath, nil
}

func isYoonSuffix(r rune) bool {
	switch r {
	case 'ゃ', 'ゅ', 'ょ', 'ぁ', 'ぃ', 'ぅ', 'ぇ', 'ぉ':
		return true
	}
	return false
}

func isConsonant(b byte) bool {
	switch b {
	case 'a', 'i', 'u', 'e', 'o':
		return false
	}
	return b >= 'a' && b <= 'z'
}

func katakanaToHiragana(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r >= 0x30A1 && r <= 0x30F6 {
			sb.WriteRune(r - 0x60)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func hiraganaToKatakana(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r >= 0x3041 && r <= 0x3096 {
			sb.WriteRune(r + 0x60)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func containsJapanese(s string) bool {
	for _, r := range s {
		if (r >= 0x3040 && r <= 0x309F) || (r >= 0x30A0 && r <= 0x30FF) || (r >= 0x4E00 && r <= 0x9FFF) {
			return true
		}
	}
	return false
}

var romajiMap = map[string]struct{ Kunrei, Hepburn string }{
	"きゃ": {"kya", "kya"}, "きゅ": {"kyu", "kyu"}, "きょ": {"kyo", "kyo"},
	"ぎゃ": {"gya", "gya"}, "ぎゅ": {"gyu", "gyu"}, "ぎょ": {"gyo", "gyo"},
	"しゃ": {"sya", "sha"}, "しゅ": {"syu", "shu"}, "しょ": {"syo", "sho"},
	"じゃ": {"zya", "ja"},  "じゅ": {"zyu", "ju"},  "じょ": {"zyo", "jo"},
	"ちゃ": {"tya", "cha"}, "ちゅ": {"tyu", "chu"}, "ちょ": {"tyo", "cho"},
	"ぢゃ": {"zya", "ja"},  "ぢゅ": {"zyu", "ju"},  "ぢょ": {"zyo", "jo"},
	"にゃ": {"nya", "nya"}, "にゅ": {"nyu", "nyu"}, "にょ": {"nyo", "nyo"},
	"ひゃ": {"hya", "hya"}, "ひゅ": {"hyu", "hyu"}, "ひょ": {"hyo", "hyo"},
	"びゃ": {"bya", "bya"}, "びゅ": {"byu", "byu"}, "びょ": {"byo", "byo"},
	"ぴゃ": {"pya", "pya"}, "ぴゅ": {"pyu", "pyu"}, "ぴょ": {"pyo", "pyo"},
	"みゃ": {"mya", "mya"}, "みゅ": {"myu", "myu"}, "みょ": {"myo", "myo"},
	"りゃ": {"rya", "rya"}, "りゅ": {"ryu", "ryu"}, "りょ": {"ryo", "ryo"},
	"ふぁ": {"fa", "fa"},   "ふぃ": {"fi", "fi"},   "ふぇ": {"fe", "fe"},   "ふぉ": {"fo", "fo"},
	"ゔぁ": {"va", "va"},   "ゔぃ": {"vi", "vi"},   "ゔぇ": {"ve", "ve"},   "ゔぉ": {"vo", "vo"},
	"ちぇ": {"tye", "che"}, "しぇ": {"sye", "she"}, "じぇ": {"zye", "je"},
	"てぃ": {"thi", "thi"}, "でぃ": {"dhi", "dhi"},
	"てゅ": {"thu", "thu"}, "でゅ": {"dhu", "dhu"},
	"とぅ": {"twu", "twu"}, "どぅ": {"dwu", "dwu"},
	"つぁ": {"tsa", "tsa"}, "つぃ": {"tsi", "tsi"}, "つぇ": {"tse", "tse"}, "つぉ": {"tso", "tso"},
	"うぃ": {"wi", "wi"}, "うぇ": {"we", "we"}, "うぉ": {"wo", "wo"},
	"いぇ": {"ye", "ye"},
	"ふゃ": {"fya", "fya"}, "ふゅ": {"fyu", "fyu"}, "ふょ": {"fyo", "fyo"},
	"ゔゃ": {"vya", "vya"}, "ゔゅ": {"vyu", "vyu"}, "ゔょ": {"vyo", "vyo"},
	"くぁ": {"kwa", "kwa"}, "くぃ": {"kwi", "kwi"}, "くぇ": {"kwe", "kwe"}, "くぉ": {"kwo", "kwo"},
	"ぐぁ": {"gwa", "gwa"}, "ぐぃ": {"gwi", "gwi"}, "ぐぇ": {"gwe", "gwe"}, "ぐぉ": {"gwo", "gwo"},

	"あ": {"a", "a"}, "い": {"i", "i"}, "う": {"u", "u"}, "え": {"e", "e"}, "お": {"o", "o"},
	"か": {"ka", "ka"}, "き": {"ki", "ki"}, "く": {"ku", "ku"}, "け": {"ke", "ke"}, "こ": {"ko", "ko"},
	"が": {"ga", "ga"}, "ぎ": {"gi", "gi"}, "ぐ": {"gu", "gu"}, "げ": {"ge", "ge"}, "ご": {"go", "go"},
	"さ": {"sa", "sa"}, "し": {"si", "shi"}, "す": {"su", "su"}, "せ": {"se", "se"}, "そ": {"so", "so"},
	"ざ": {"za", "za"}, "じ": {"zi", "ji"}, "ず": {"zu", "zu"}, "ぜ": {"ze", "ze"}, "ぞ": {"zo", "zo"},
	"た": {"ta", "ta"}, "ち": {"ti", "chi"}, "つ": {"tu", "tsu"}, "て": {"te", "te"}, "と": {"to", "to"},
	"だ": {"da", "da"}, "ぢ": {"zi", "ji"}, "づ": {"zu", "zu"}, "で": {"de", "de"}, "ど": {"do", "do"},
	"な": {"na", "na"}, "に": {"ni", "ni"}, "ぬ": {"nu", "nu"}, "ね": {"ne", "ne"}, "の": {"no", "no"},
	"は": {"ha", "ha"}, "ひ": {"hi", "hi"}, "ふ": {"hu", "fu"}, "へ": {"he", "he"}, "ほ": {"ho", "ho"},
	"ば": {"ba", "ba"}, "び": {"bi", "bi"}, "ぶ": {"bu", "bu"}, "べ": {"be", "be"}, "ぼ": {"bo", "bo"},
	"ぱ": {"pa", "pa"}, "ぴ": {"pi", "pi"}, "ぷ": {"pu", "pu"}, "ぺ": {"pe", "pe"}, "ぽ": {"po", "po"},
	"ま": {"ma", "ma"}, "み": {"mi", "mi"}, "む": {"mu", "mu"}, "め": {"me", "me"}, "も": {"mo", "mo"},
	"や": {"ya", "ya"}, "ゆ": {"yu", "yu"}, "よ": {"yo", "yo"},
	"ら": {"ra", "ra"}, "り": {"ri", "ri"}, "る": {"ru", "ru"}, "れ": {"re", "re"}, "ろ": {"ro", "ro"},
	"わ": {"wa", "wa"}, "を": {"o", "wo"}, "ん": {"n", "n"},
	"ゔ": {"vu", "vu"},
	"ゐ": {"i", "i"}, "ゑ": {"e", "e"},
	"ぁ": {"la", "la"}, "ぃ": {"li", "li"}, "ぅ": {"lu", "lu"}, "ぇ": {"le", "le"}, "ぉ": {"lo", "lo"},
	"ゃ": {"lya", "lya"}, "ゅ": {"lyu", "lyu"}, "ょ": {"lyo", "lyo"},
	"っ": {"ltu", "ltu"},
	"ゎ": {"lwa", "lwa"},
	"ゕ": {"lka", "lka"}, "ゖ": {"lke", "lke"},
}

func hiraganaToRomaji(input string) (kunrei string, hepburn string) {
	normalized := katakanaToHiragana(input)
	var kResult strings.Builder
	var hResult strings.Builder

	runes := []rune(normalized)
	i := 0
	n := len(runes)

	for i < n {
		if runes[i] == 'っ' {
			doubled := false
			if i+1 < n {
				nextRune := runes[i+1]
				nextStr := string(nextRune)
				if i+2 < n && isYoonSuffix(runes[i+2]) {
					nextStr = string(runes[i+1 : i+3])
				}

				kNext, hNext := lookupRomaji(nextStr)

				kConsonant := len(kNext) > 0 && isConsonant(kNext[0])
				hConsonant := len(hNext) > 0 && isConsonant(hNext[0])

				if kConsonant {
					kResult.WriteByte(kNext[0])
				}
				if hConsonant {
					if strings.HasPrefix(hNext, "ch") {
						hResult.WriteByte('t')
					} else {
						hResult.WriteByte(hNext[0])
					}
				}
				if kConsonant || hConsonant {
					doubled = true
				}
			}
			if !doubled {
				kResult.WriteString("ltu")
				hResult.WriteString("ltu")
			}
			i++
			continue
		}

		if runes[i] == 'ー' {
			i++
			continue
		}

		if i+1 < n {
			twoChars := string(runes[i : i+2])
			if val, ok := romajiMap[twoChars]; ok {
				kResult.WriteString(val.Kunrei)
				hResult.WriteString(val.Hepburn)
				i += 2
				continue
			}
		}

		charStr := string(runes[i])
		if val, ok := romajiMap[charStr]; ok {
			kResult.WriteString(val.Kunrei)
			hResult.WriteString(val.Hepburn)
		} else {
			kResult.WriteRune(runes[i])
			hResult.WriteRune(runes[i])
		}
		i++
	}

	return kResult.String(), hResult.String()
}

func lookupRomaji(s string) (string, string) {
	if val, ok := romajiMap[s]; ok {
		return val.Kunrei, val.Hepburn
	}
	return "", ""
}

func addUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}

func createEmojiZip(zipPath string, items []zipItem, entries []MisskeyEmojiEntry) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	// 1. Create meta.json
	meta := MisskeyMeta{
		MetaVersion: 2,
		ExportedAt:  time.Now().UTC().Format(time.RFC3339),
		Emojis:      entries,
	}
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal meta.json: %w", err)
	}

	metaFileWriter, err := archive.Create("meta.json")
	if err != nil {
		return fmt.Errorf("failed to create meta.json in zip: %w", err)
	}
	if _, err := metaFileWriter.Write(metaBytes); err != nil {
		return fmt.Errorf("failed to write meta.json to zip: %w", err)
	}

	// 2. Add each image
	for _, item := range items {
		imgFile, err := os.Open(item.absPath)
		if err != nil {
			return fmt.Errorf("failed to open processed image %s: %w", item.absPath, err)
		}

		imgFileWriter, err := archive.Create(item.fileName)
		if err != nil {
			imgFile.Close()
			return fmt.Errorf("failed to create file %s in zip: %w", item.fileName, err)
		}

		if _, err := io.Copy(imgFileWriter, imgFile); err != nil {
			imgFile.Close()
			return fmt.Errorf("failed to copy file %s to zip: %w", item.fileName, err)
		}
		imgFile.Close()
	}

	return nil
}

func isPureHiraganaOrSafe(s string) bool {
	for _, r := range s {
		if (r >= 0x3041 && r <= 0x3096) || r == 0x3094 || r == 'ー' {
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}
