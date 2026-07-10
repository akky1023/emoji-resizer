package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
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
	_, err = processImage(srcPath, outDir1, 128, "", false, false, "")
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
	_, err = processImage(srcPath, outDir2, 128, "", false, true, "")
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
	_, err = processImage(srcPath3, outDir3, 128, "", false, true, "")
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
		{"cat", true},  // English is considered safe too (containsJapanese will be false, so it won't prompt either)
	}

	for _, tc := range tests {
		got := isPureHiraganaOrSafe(tc.input)
		if got != tc.expected {
			t.Errorf("isPureHiraganaOrSafe(%q) = %t; expected %t", tc.input, got, tc.expected)
		}
	}
}
