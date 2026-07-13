package main

import (
	"bufio"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessImageRect(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "emoji-resizer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a 100x200 (width x height) test image (vertical rectangle)
	w, h := 100, 200
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Fill with some pattern or just keep blank (transparent)
	srcPath := filepath.Join(tmpDir, "test_vertical.png")
	f, err := os.Create(srcPath)
	if err != nil {
		t.Fatalf("failed to create source image file: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode source image: %v", err)
	}
	f.Close()

	// Test case 1: Standard square (padding) resize (rect = false)
	// Target size: 128x128
	// Output should be a 128x128 square image.
	outDir1 := filepath.Join(tmpDir, "out1")
	_, _, err = processImage(srcPath, outDir1, 128, "", false, false, "", false, 0, false)
	if err != nil {
		t.Errorf("processImage (rect=false) failed: %v", err)
	}
	outPath1 := filepath.Join(outDir1, "test_vertical.png")
	f1, err := os.Open(outPath1)
	if err != nil {
		t.Fatalf("failed to open output image 1: %v", err)
	}
	img1, err := png.Decode(f1)
	f1.Close()
	if err != nil {
		t.Fatalf("failed to decode output image 1: %v", err)
	}
	b1 := img1.Bounds()
	if b1.Dx() != 128 || b1.Dy() != 128 {
		t.Errorf("expected 128x128 image, got %dx%d", b1.Dx(), b1.Dy())
	}

	// Test case 2: Rect mode resize (rect = true)
	// Target size: 128 (short side matches 128)
	// Since w < h (100 < 200), width (short side) should become 128, and height should scale to 256.
	outDir2 := filepath.Join(tmpDir, "out2")
	_, _, err = processImage(srcPath, outDir2, 128, "", false, true, "", false, 0, false)
	if err != nil {
		t.Errorf("processImage (rect=true, vertical) failed: %v", err)
	}
	outPath2 := filepath.Join(outDir2, "test_vertical.png")
	f2, err := os.Open(outPath2)
	if err != nil {
		t.Fatalf("failed to open output image 2: %v", err)
	}
	img2, err := png.Decode(f2)
	f2.Close()
	if err != nil {
		t.Fatalf("failed to decode output image 2: %v", err)
	}
	b2 := img2.Bounds()
	if b2.Dx() != 128 || b2.Dy() != 256 {
		t.Errorf("expected 128x256 image, got %dx%d", b2.Dx(), b2.Dy())
	}

	// Create a 300x150 (width x height) test image (horizontal rectangle)
	w3, h3 := 300, 150
	img3 := image.NewRGBA(image.Rect(0, 0, w3, h3))
	srcPath3 := filepath.Join(tmpDir, "test_horizontal.png")
	f3, err := os.Create(srcPath3)
	if err != nil {
		t.Fatalf("failed to create source image file 3: %v", err)
	}
	if err := png.Encode(f3, img3); err != nil {
		f3.Close()
		t.Fatalf("failed to encode source image 3: %v", err)
	}
	f3.Close()

	// Test case 3: Rect mode resize on horizontal image
	// Target size: 128 (short side matches 128)
	// Since h3 < w3 (150 < 300), height (short side) should become 128, and width should scale to 256.
	outDir3 := filepath.Join(tmpDir, "out3")
	_, _, err = processImage(srcPath3, outDir3, 128, "", false, true, "", false, 0, false)
	if err != nil {
		t.Errorf("processImage (rect=true, horizontal) failed: %v", err)
	}
	outPath3 := filepath.Join(outDir3, "test_horizontal.png")
	f4, err := os.Open(outPath3)
	if err != nil {
		t.Fatalf("failed to open output image 3: %v", err)
	}
	img4, err := png.Decode(f4)
	f4.Close()
	if err != nil {
		t.Fatalf("failed to decode output image 3: %v", err)
	}
	b3 := img4.Bounds()
	if b3.Dx() != 256 || b3.Dy() != 128 {
		t.Errorf("expected 256x128 image, got %dx%d", b3.Dx(), b3.Dy())
	}
}

func TestRomajiConversion(t *testing.T) {
	tests := []struct {
		input   string
		kunrei  string
		hepburn string
	}{
		{"ねこ", "neko", "neko"},
		{"ネコ", "neko", "neko"},
		{"しんぶん", "sinbun", "shinbun"},
		{"がっこう", "gakkou", "gakkou"},
		{"まっちゃ", "mattya", "matcha"},
		{"らーめん", "ramen", "ramen"},
		{"しゃしん", "syasin", "shashin"},
		{"かんじ", "kanzi", "kanji"},
		{"てすと_123", "tesuto_123", "tesuto_123"},
		{"ぃ", "li", "xi"},
		{"ぁ", "la", "xa"},
		{"がっ", "galtu", "gaxtu"},
		{"っっ", "ltultu", "xtuxtu"},
		{"っぃ", "ltuli", "xtuxi"},
		{"ふぃっしゅ", "fissyu", "fisshu"},
		{"ぇ", "le", "xe"},
		{"ヶ", "lke", "xke"},
		{"ヵ", "lka", "xka"},
		{"てぃ", "thi", "thi"},
		{"でぃ", "dhi", "dhi"},
		{"とぅ", "twu", "twu"},
		{"どぅ", "dwu", "dwu"},
		{"てゅ", "thu", "thu"},
		{"でゅ", "dhu", "dhu"},
		{"うぃ", "wi", "wi"},
		{"ちぇ", "tye", "che"},
		{"しぇ", "sye", "she"},
		{"じぇ", "zye", "je"},
	}

	for _, tc := range tests {
		k, h := hiraganaToRomaji(tc.input)
		if k != tc.kunrei {
			t.Errorf("hiraganaToRomaji(%q) Kunrei: expected %q, got %q", tc.input, tc.kunrei, k)
		}
		if h != tc.hepburn {
			t.Errorf("hiraganaToRomaji(%q) Hepburn: expected %q, got %q", tc.input, tc.hepburn, h)
		}
	}
}

func TestKatakanaToHiragana(t *testing.T) {
	input := "テストラーメン"
	expected := "てすとらーめん"
	got := katakanaToHiragana(input)
	if got != expected {
		t.Errorf("katakanaToHiragana(%q) = %q; expected %q", input, got, expected)
	}
}

func TestHiraganaToKatakana(t *testing.T) {
	input := "てすとらーめん"
	expected := "テストラーメン"
	got := hiraganaToKatakana(input)
	if got != expected {
		t.Errorf("hiraganaToKatakana(%q) = %q; expected %q", input, got, expected)
	}
}

func TestContainsJapanese(t *testing.T) {
	if !containsJapanese("ねこ") {
		t.Errorf("expected containsJapanese(\"ねこ\") to be true")
	}
	if !containsJapanese("ネコ") {
		t.Errorf("expected containsJapanese(\"ネコ\") to be true")
	}
	if !containsJapanese("猫") {
		t.Errorf("expected containsJapanese(\"猫\") to be true")
	}
	if containsJapanese("cat_123-") {
		t.Errorf("expected containsJapanese(\"cat_123-\") to be false")
	}
}

func TestIsPureHiraganaOrSafe(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"ねこ", true},
		{"ねこ123_-", true},
		{"らーめん", true},
		{"ネコ", false}, // Katakana
		{"猫", false},  // Kanji
		{"cat", false}, // Pure English is not hiragana, and should not generate aliases
	}

	for _, tc := range tests {
		got := isPureHiraganaOrSafe(tc.input)
		if got != tc.expected {
			t.Errorf("isPureHiraganaOrSafe(%q) = %t; expected %t", tc.input, got, tc.expected)
		}
	}
}

func TestProcessImageAutoRect(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "emoji-resizer-autorect-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Case 1: 100x120 (aspect ratio = 1.2 < 1.618 golden ratio).
	// Under default auto-rect, it should NOT trigger rect mode -> processed as square (padding to 128x128).
	img1 := image.NewRGBA(image.Rect(0, 0, 100, 120))
	srcPath1 := filepath.Join(tmpDir, "img1.png")
	f1, _ := os.Create(srcPath1)
	png.Encode(f1, img1)
	f1.Close()

	outDir1 := filepath.Join(tmpDir, "out1")
	outPath1, _, err := processImage(srcPath1, outDir1, 128, "", false, false, "", true, 0, false)
	if err != nil {
		t.Fatalf("failed to process image 1: %v", err)
	}
	rf1, _ := os.Open(outPath1)
	ri1, _ := png.Decode(rf1)
	rf1.Close()
	if ri1.Bounds().Dx() != 128 || ri1.Bounds().Dy() != 128 {
		t.Errorf("expected 128x128 for ratio 1.2 under golden ratio thres, got %dx%d", ri1.Bounds().Dx(), ri1.Bounds().Dy())
	}

	// Case 2: 100x200 (aspect ratio = 2.0 > 1.618 golden ratio).
	// Under default auto-rect, it SHOULD trigger rect mode -> kept as rect (short side 128, long side 256).
	img2 := image.NewRGBA(image.Rect(0, 0, 100, 200))
	srcPath2 := filepath.Join(tmpDir, "img2.png")
	f2, _ := os.Create(srcPath2)
	png.Encode(f2, img2)
	f2.Close()

	outDir2 := filepath.Join(tmpDir, "out2")
	outPath2, _, err := processImage(srcPath2, outDir2, 128, "", false, false, "", true, 0, false)
	if err != nil {
		t.Fatalf("failed to process image 2: %v", err)
	}
	rf2, _ := os.Open(outPath2)
	ri2, _ := png.Decode(rf2)
	rf2.Close()
	if ri2.Bounds().Dx() != 128 || ri2.Bounds().Dy() != 256 {
		t.Errorf("expected 128x256 for ratio 2.0 under golden ratio thres, got %dx%d", ri2.Bounds().Dx(), ri2.Bounds().Dy())
	}

	// Case 3: 100x200 (ratio 2.0) with custom ratio threshold = 2.5 (2.0 < 2.5).
	// It should NOT trigger rect mode -> processed as square (128x128).
	outDir3 := filepath.Join(tmpDir, "out3")
	outPath3, _, err := processImage(srcPath2, outDir3, 128, "", false, false, "", true, 2.5, false)
	if err != nil {
		t.Fatalf("failed to process image 3: %v", err)
	}
	rf3, _ := os.Open(outPath3)
	ri3, _ := png.Decode(rf3)
	rf3.Close()
	if ri3.Bounds().Dx() != 128 || ri3.Bounds().Dy() != 128 {
		t.Errorf("expected 128x128 for ratio 2.0 under 2.5 thres, got %dx%d", ri3.Bounds().Dx(), ri3.Bounds().Dy())
	}

	// Case 4: 100x200 (ratio 2.0) with custom ratio threshold = 1.5 (2.0 > 1.5).
	// It SHOULD trigger rect mode -> processed as rect (128x256).
	outDir4 := filepath.Join(tmpDir, "out4")
	outPath4, _, err := processImage(srcPath2, outDir4, 128, "", false, false, "", true, 1.5, false)
	if err != nil {
		t.Fatalf("failed to process image 4: %v", err)
	}
	rf4, _ := os.Open(outPath4)
	ri4, _ := png.Decode(rf4)
	rf4.Close()
	if ri4.Bounds().Dx() != 128 || ri4.Bounds().Dy() != 256 {
		t.Errorf("expected 128x256 for ratio 2.0 under 1.5 thres, got %dx%d", ri4.Bounds().Dx(), ri4.Bounds().Dy())
	}
}

func TestRecursiveZipMode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "emoji-resizer-zip-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectories
	subDir1 := filepath.Join(tmpDir, "sub1")
	subDir2 := filepath.Join(tmpDir, "sub1", "sub2")
	if err := os.MkdirAll(subDir2, 0755); err != nil {
		t.Fatalf("failed to create subdirs: %v", err)
	}

	// Create dummy images in subdirectories
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img1Path := filepath.Join(subDir1, "img1.png")
	f1, _ := os.Create(img1Path)
	png.Encode(f1, img)
	f1.Close()

	img2Path := filepath.Join(subDir2, "img2.png")
	f2, _ := os.Create(img2Path)
	png.Encode(f2, img)
	f2.Close()

	// Scan directory recursively
	filesToProcess, err := scanDirectory(tmpDir, true, "", "")
	if err != nil {
		t.Fatalf("scanDirectory failed: %v", err)
	}

	if len(filesToProcess) != 2 {
		t.Fatalf("expected 2 files, got %d", len(filesToProcess))
	}

	// Determine topLevelOutDir (same logic as in main)
	topLevelOutDir := filepath.Join(tmpDir, "output")
	absTopLevelInDir, _ := filepath.Abs(tmpDir)

	// Process files and collect zip items grouped by directory
	dirZips := make(map[string]*dirZipData)
	var allZipItems []zipItem
	var allEmojiEntries []MisskeyEmojiEntry

	for _, filePath := range filesToProcess {
		destPath, _, err := processImage(filePath, "", 128, "", false, false, "", false, 0, false)
		if err != nil {
			t.Fatalf("processImage failed for %s: %v", filePath, err)
		}
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

		entry := MisskeyEmojiEntry{
			FileName:   outFileName,
			Downloaded: true,
			Emoji: MisskeyEmojiInfo{
				Name: "test",
			},
		}
		data.entries = append(data.entries, entry)
		allEmojiEntries = append(allEmojiEntries, entry)
	}

	categoryPrefix := "testcat"

	// Create local emojis.zip (with suffix) for each directory in topLevelOutDir
	for _, data := range dirZips {
		if err := os.MkdirAll(topLevelOutDir, 0755); err != nil {
			t.Fatalf("failed to create topLevelOutDir: %v", err)
		}
		zipPath := filepath.Join(topLevelOutDir, categoryPrefix+"_"+data.suffixName+".zip")
		err = createEmojiZip(zipPath, data.items, data.entries)
		if err != nil {
			t.Fatalf("createEmojiZip failed for local: %v", err)
		}
		if _, err := os.Stat(zipPath); os.IsNotExist(err) {
			t.Errorf("expected local emojis.zip to exist at %s, but it does not", zipPath)
		}
	}

	// Create top-level testcat_all.zip
	if err := os.MkdirAll(topLevelOutDir, 0755); err != nil {
		t.Fatalf("failed to create topLevelOutDir: %v", err)
	}
	allemojiPath := filepath.Join(topLevelOutDir, categoryPrefix+"_all.zip")
	err = createEmojiZip(allemojiPath, allZipItems, allEmojiEntries)
	if err != nil {
		t.Fatalf("createEmojiZip failed for allemoji: %v", err)
	}

	// Check if testcat_all.zip exists
	if _, err := os.Stat(allemojiPath); os.IsNotExist(err) {
		t.Errorf("expected testcat_all.zip to exist at %s, but it does not", allemojiPath)
	}

	// Verify that the specific files are created
	sub1Zip := filepath.Join(topLevelOutDir, "testcat_sub1.zip")
	sub2Zip := filepath.Join(topLevelOutDir, "testcat_sub1_sub2.zip")
	if _, err := os.Stat(sub1Zip); os.IsNotExist(err) {
		t.Errorf("expected testcat_sub1.zip to exist, but it does not")
	}
	if _, err := os.Stat(sub2Zip); os.IsNotExist(err) {
		t.Errorf("expected testcat_sub1_sub2.zip to exist, but it does not")
	}
}

func TestProcessImageSkipExist(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "emoji-resizer-skip-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	w, h := 100, 100
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	srcPath := filepath.Join(tmpDir, "test.png")
	f, err := os.Create(srcPath)
	if err != nil {
		t.Fatalf("failed to create source image file: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode source image: %v", err)
	}
	f.Close()

	// 1. Process with skipExist = false (should create the output file)
	outDir := filepath.Join(tmpDir, "out")
	outPath, skipped, err := processImage(srcPath, outDir, 128, "", false, false, "", false, 0, false)
	if err != nil {
		t.Fatalf("first processImage failed: %v", err)
	}
	if skipped {
		t.Errorf("expected skipped to be false on first run")
	}

	// Modify the created output file to check if it gets overwritten
	testBytes := []byte("dummy modified content")
	if err := os.WriteFile(outPath, testBytes, 0644); err != nil {
		t.Fatalf("failed to write dummy content to output file: %v", err)
	}

	// 2. Process with skipExist = true (should skip and NOT overwrite)
	_, skipped2, err := processImage(srcPath, outDir, 128, "", false, false, "", false, 0, true)
	if err != nil {
		t.Fatalf("second processImage failed: %v", err)
	}
	if !skipped2 {
		t.Errorf("expected skipped to be true on second run")
	}

	// Verify content remains "dummy modified content" (not overwritten by resized image)
	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(content) != "dummy modified content" {
		t.Errorf("file was overwritten: expected 'dummy modified content', got %q", string(content))
	}
}

func TestNamePrefixSuffixZipMode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "emoji-resizer-prefix-suffix-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create dummy image with Japanese name "ねこ.png"
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	imgName := "ねこ.png"
	imgPath := filepath.Join(tmpDir, imgName)
	f, err := os.Create(imgPath)
	if err != nil {
		t.Fatalf("failed to create source image: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode source image: %v", err)
	}
	f.Close()

	// Settings for the prefix and suffix
	namePrefix := "pref_"
	nameSuffix := "_suf"

	// Mimic the main.go processing logic for zipMode
	ext := filepath.Ext(imgPath)
	base := strings.TrimSuffix(filepath.Base(imgPath), ext)

	var name, hiragana, katakana, hepburn string
	var hasPronunciation bool

	if isPureHiraganaOrSafe(base) {
		hiragana = base
		katakana = hiraganaToKatakana(hiragana)
		var hepburnRaw string
		name, hepburnRaw = hiraganaToRomaji(hiragana)
		name = strings.ToLower(name)
		hepburn = strings.ToLower(hepburnRaw)
		hasPronunciation = true
	} else {
		t.Fatalf("expected 'ねこ' to be pure hiragana")
	}

	customBase := namePrefix + name + nameSuffix

	// Run processImage
	destPath, _, err := processImage(imgPath, tmpDir, 128, "", false, false, customBase, false, 0, false)
	if err != nil {
		t.Fatalf("processImage failed: %v", err)
	}

	outFileName := filepath.Base(destPath)
	expectedFileName := "pref_neko_suf.png"
	if outFileName != expectedFileName {
		t.Errorf("expected file name %q, got %q", expectedFileName, outFileName)
	}

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
			Name:    namePrefix + name + nameSuffix,
			Aliases: aliases,
		},
	}

	expectedName := "pref_neko_suf"
	if entry.Emoji.Name != expectedName {
		t.Errorf("expected emoji name %q, got %q", expectedName, entry.Emoji.Name)
	}

	if len(entry.Emoji.Aliases) != 2 {
		t.Errorf("expected 2 aliases, got %d", len(entry.Emoji.Aliases))
	}
	expectedAliases := []string{"ねこ", "ネコ"}
	for i, alias := range expectedAliases {
		if i < len(entry.Emoji.Aliases) && entry.Emoji.Aliases[i] != alias {
			t.Errorf("expected alias %q, got %q", alias, entry.Emoji.Aliases[i])
		}
	}
}

func TestParseAndApplyConfig(t *testing.T) {
	// Create temporary directory and configuration file
	tmpDir, err := os.MkdirTemp("", "emoji-resizer-config-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configContent := `{
		"size": 256,
		"out": "config_output",
		"suffix": "_configured",
		"name_prefix": "conf_pref_",
		"name_suffix": "_conf_suf",
		"r": true,
		"no_resize": true,
		"rect": true,
		"zip": true,
		"skip": true,
		"category": "ConfigCat",
		"license": "ConfigLic",
		"auto_rect": 2.5
	}`

	cfgPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(cfgPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// 1. Test standard parsing and application (no seen flags)
	size := 128
	outDir := ""
	suffix := ""
	namePrefix := ""
	nameSuffix := ""
	recursive := false
	noResize := false
	rect := false
	zipMode := false
	skip := false
	var autoRect AutoRectValue
	var cfgCategory string
	var cfgLicense string

	seenFlags := make(map[string]bool)

	err = parseAndApplyConfig(cfgPath, seenFlags, &size, &outDir, &suffix, &namePrefix, &nameSuffix, &recursive, &noResize, &rect, &zipMode, &skip, &autoRect, &cfgCategory, &cfgLicense)
	if err != nil {
		t.Fatalf("parseAndApplyConfig failed: %v", err)
	}

	if size != 256 {
		t.Errorf("expected size 256, got %d", size)
	}
	if outDir != "config_output" {
		t.Errorf("expected outDir 'config_output', got %q", outDir)
	}
	if suffix != "_configured" {
		t.Errorf("expected suffix '_configured', got %q", suffix)
	}
	if namePrefix != "conf_pref_" {
		t.Errorf("expected namePrefix 'conf_pref_', got %q", namePrefix)
	}
	if nameSuffix != "_conf_suf" {
		t.Errorf("expected nameSuffix '_conf_suf', got %q", nameSuffix)
	}
	if !recursive {
		t.Errorf("expected recursive to be true")
	}
	if !noResize {
		t.Errorf("expected noResize to be true")
	}
	if !rect {
		t.Errorf("expected rect to be true")
	}
	if !zipMode {
		t.Errorf("expected zipMode to be true")
	}
	if !skip {
		t.Errorf("expected skip to be true")
	}
	if cfgCategory != "ConfigCat" {
		t.Errorf("expected cfgCategory 'ConfigCat', got %q", cfgCategory)
	}
	if cfgLicense != "ConfigLic" {
		t.Errorf("expected cfgLicense 'ConfigLic', got %q", cfgLicense)
	}
	if !autoRect.Active || autoRect.Ratio != 2.5 {
		t.Errorf("expected autoRect to be active and 2.5, got Active=%t, Ratio=%f", autoRect.Active, autoRect.Ratio)
	}

	// 2. Test seen flags override (CLI args should override config settings)
	// Reset variables
	size = 128
	outDir = ""
	seenFlags = map[string]bool{
		"size": true,
		"out":  true,
	}

	err = parseAndApplyConfig(cfgPath, seenFlags, &size, &outDir, &suffix, &namePrefix, &nameSuffix, &recursive, &noResize, &rect, &zipMode, &skip, &autoRect, &cfgCategory, &cfgLicense)
	if err != nil {
		t.Fatalf("parseAndApplyConfig override run failed: %v", err)
	}

	// size and outDir should remain at 128 and "" (not overridden by config because they are in seenFlags)
	if size != 128 {
		t.Errorf("expected size 128 (not overridden by config), got %d", size)
	}
	if outDir != "" {
		t.Errorf("expected outDir '' (not overridden by config), got %q", outDir)
	}
}

func TestComputeEmojiName(t *testing.T) {
	// Test computeEmojiName without zipMode
	customBase, name, hiragana, katakana, hepburn, hasPronunciation := computeEmojiName("path/to/ねこ.png", false, "pref_", "_suff", nil)
	if customBase != "pref_ねこ_suff" {
		t.Errorf("expected customBase pref_ねこ_suff, got %s", customBase)
	}
	_ = name
	_ = hiragana
	_ = katakana
	_ = hepburn
	_ = hasPronunciation

	// Test computeEmojiName with zipMode (pure hiragana)
	customBase, name, hiragana, katakana, hepburn, hasPronunciation = computeEmojiName("path/to/ねこ.png", true, "pref_", "_suff", nil)
	if customBase != "pref_neko_suff" {
		t.Errorf("expected customBase pref_neko_suff, got %s", customBase)
	}
	if name != "neko" || hiragana != "ねこ" || katakana != "ネコ" || hepburn != "neko" || !hasPronunciation {
		t.Errorf("unexpected outputs: name=%q, hiragana=%q, katakana=%q, hepburn=%q, hasPronunciation=%t", name, hiragana, katakana, hepburn, hasPronunciation)
	}

	// Test computeEmojiName with zipMode (contains Japanese, requiring prompt)
	inputReader := bufio.NewReader(strings.NewReader("いぬ\n"))
	customBase2, name2, hiragana2, katakana2, hepburn2, hasPronunciation2 := computeEmojiName("path/to/犬.png", true, "pref_", "_suff", inputReader)
	if customBase2 != "pref_inu_suff" {
		t.Errorf("expected customBase pref_inu_suff, got %s", customBase2)
	}
	if name2 != "inu" || hiragana2 != "いぬ" || katakana2 != "イヌ" || hepburn2 != "inu" || !hasPronunciation2 {
		t.Errorf("unexpected outputs: name=%q, hiragana=%q, katakana=%q, hepburn=%q, hasPronunciation=%t", name2, hiragana2, katakana2, hepburn2, hasPronunciation2)
	}
}

func TestCheckModeIntegration(t *testing.T) {
	binaryPath := filepath.Join("..", "emoji-resizer.exe")
	// Rebuild the binary to ensure it has our latest code
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}

	// Create temp directory for testing files
	tmpDir, err := os.MkdirTemp("", "emoji-resizer-check-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dirA := filepath.Join(tmpDir, "dirA")
	dirB := filepath.Join(tmpDir, "dirB")
	if err := os.MkdirAll(dirA, 0755); err != nil {
		t.Fatalf("failed to create dirA: %v", err)
	}
	if err := os.MkdirAll(dirB, 0755); err != nil {
		t.Fatalf("failed to create dirB: %v", err)
	}

	// Write simple PNG files
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	
	writeDummyPNG := func(path string) {
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("failed to create dummy file %s: %v", path, err)
		}
		defer f.Close()
		if err := png.Encode(f, img); err != nil {
			t.Fatalf("failed to encode dummy png %s: %v", path, err)
		}
	}

	fileA1 := filepath.Join(dirA, "apple.png")
	fileA2 := filepath.Join(dirA, "banana.png")
	fileB1 := filepath.Join(dirB, "apple.png")
	fileB2 := filepath.Join(dirB, "orange.png")

	writeDummyPNG(fileA1)
	writeDummyPNG(fileA2)
	writeDummyPNG(fileB1)
	writeDummyPNG(fileB2)

	// Clean paths to match what we expect in the stdout
	cleanFileA1 := filepath.Clean(fileA1)
	cleanFileB1 := filepath.Clean(fileB1)

	// Test Case 1: Run with duplicate names (apple.png in both dirA and dirB)
	cmdCheckDup := exec.Command(binaryPath, "-check", fileA1, fileA2, fileB1, fileB2)
	outBytes, err := cmdCheckDup.Output()
	// Exit code should be 1 because duplicates exist
	if err == nil {
		t.Errorf("expected exit status 1 for duplicate check, got exit status 0 (stdout: %q)", string(outBytes))
	} else {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 1 {
				t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
			}
		} else {
			t.Fatalf("unexpected execution error: %v", err)
		}
	}

	outStr := strings.TrimSpace(string(outBytes))
	// Normalize line endings to \n for comparison
	outStr = strings.ReplaceAll(outStr, "\r\n", "\n")
	expectedDupOutput := cleanFileA1 + "\n" + cleanFileB1
	if outStr != expectedDupOutput {
		t.Errorf("expected duplicate output:\n%q\nbut got:\n%q", expectedDupOutput, outStr)
	}

	// Test Case 2: Run without duplicate names (only unique files)
	cmdCheckOK := exec.Command(binaryPath, "-check", fileA1, fileA2, fileB2)
	outBytesOK, err := cmdCheckOK.Output()
	if err != nil {
		t.Errorf("expected exit status 0 for check without duplicates, got error: %v (stdout: %q)", err, string(outBytesOK))
	}
	outStrOK := strings.TrimSpace(string(outBytesOK))
	if outStrOK != "OK" {
		t.Errorf("expected stdout to be 'OK', got %q", outStrOK)
	}

	// Test Case 3: Run with mixed normal and ZIP names (emozi.png and えもじ.png)
	fileA3 := filepath.Join(dirA, "emozi.png")
	fileB3 := filepath.Join(dirB, "えもじ.png") // hiragana
	writeDummyPNG(fileA3)
	writeDummyPNG(fileB3)

	cleanFileA3 := filepath.Clean(fileA3)
	cleanFileB3 := filepath.Clean(fileB3)

	cmdCheckMixed := exec.Command(binaryPath, "-check", fileA3, fileB3)
	outBytesMixed, err := cmdCheckMixed.Output()
	if err == nil {
		t.Errorf("expected exit status 1 for mixed duplicate check, got exit status 0 (stdout: %q)", string(outBytesMixed))
	} else {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 1 {
				t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
			}
		} else {
			t.Fatalf("unexpected execution error: %v", err)
		}
	}

	outStrMixed := strings.TrimSpace(string(outBytesMixed))
	outStrMixed = strings.ReplaceAll(outStrMixed, "\r\n", "\n")
	expectedMixedOutput := cleanFileA3 + "\n" + cleanFileB3
	if outStrMixed != expectedMixedOutput {
		t.Errorf("expected mixed duplicate output:\n%q\nbut got:\n%q", expectedMixedOutput, outStrMixed)
	}
}

func TestOptionalConfigFlagAndFile(t *testing.T) {
	binaryPath := filepath.Join("..", "emoji-resizer.exe")
	// Rebuild the binary to ensure it has our latest code
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}

	// Create temp directory for testing files
	tmpDir, err := os.MkdirTemp("", "emoji-resizer-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a subfolder inside tmpDir that acts as the working directory
	workingDir := filepath.Join(tmpDir, "work")
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		t.Fatalf("failed to create workingDir: %v", err)
	}

	// Create a dummy image inside workingDir
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	imgFile := filepath.Join(workingDir, "test.png")
	f, err := os.Create(imgFile)
	if err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}
	png.Encode(f, img)
	f.Close()

	// Write a config file named 'config' (without extension) that changes the name prefix/suffix
	configContent := `{"name_prefix": "pre_", "name_suffix": "_suf"}`
	configFile := filepath.Join(workingDir, "config")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// 1. Run without specifying -config but with -check
	// Since 'config' exists, it should automatically load it and use prefix 'pre_' and suffix '_suf'
	// So emoji name should be 'pre_test_suf'
	absBinaryPath, err := filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("failed to get absolute path of binary: %v", err)
	}

	cmdRun := exec.Command(absBinaryPath, "-check", imgFile)
	cmdRun.Dir = workingDir
	outBytes, err := cmdRun.Output()
	if err != nil {
		t.Fatalf("execution failed: %v, stdout: %q", err, string(outBytes))
	}
	if strings.TrimSpace(string(outBytes)) != "OK" {
		t.Errorf("expected OK, got %q", string(outBytes))
	}

	// 2. Run with -config flag but with no argument value
	// We want to verify it doesn't fail with "flag needs an argument: -config"
	cmdRun2 := exec.Command(absBinaryPath, "-config", "-check", imgFile)
	cmdRun2.Dir = workingDir
	outBytes2, err := cmdRun2.Output()
	if err != nil {
		t.Fatalf("execution with empty -config failed: %v, stdout: %q", err, string(outBytes2))
	}
	if strings.TrimSpace(string(outBytes2)) != "OK" {
		t.Errorf("expected OK, got %q", string(outBytes2))
	}
}



