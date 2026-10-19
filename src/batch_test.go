package main

import (
	"bufio"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParallelBatchCollisions(t *testing.T) {
	for _, skip := range []bool{false, true} {
		t.Run(map[bool]string{false: "overwrite", true: "skip"}[skip], func(t *testing.T) {
			root := t.TempDir()
			var files []string
			for i, name := range []string{"a~same.png", "b~same.png", "unique.png"} {
				path := filepath.Join(root, name)
				img := image.NewRGBA(image.Rect(0, 0, 8, 8))
				img.Set(0, 0, color.RGBA{R: uint8(50 + i*50), A: 255})
				f, err := os.Create(path)
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(f, img)
				f.Close()
				if err != nil {
					t.Fatal(err)
				}
				files = append(files, path)
			}
			bad := filepath.Join(root, "broken.png")
			if err := os.WriteFile(bad, []byte("invalid"), 0644); err != nil {
				t.Fatal(err)
			}
			files = append(files, bad)
			opts := &appOptions{jobs: 4, zipMode: true, noResize: true, skip: skip, outDir: filepath.Join(root, "out")}
			success, failure, _, items, entries := processBatchImages(files, opts, bufio.NewReader(strings.NewReader("")), root)
			if success != 3 || failure != 1 || len(items) != 3 || len(entries) != 3 {
				t.Fatalf("counts: %d successes, %d failures, %d items, %d entries", success, failure, len(items), len(entries))
			}
			for i, category := range []string{"a", "b", ""} {
				if entries[i].Emoji.Category != category {
					t.Fatalf("metadata order changed: %+v", entries)
				}
			}
			f, err := os.Open(filepath.Join(opts.outDir, "same.png"))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			img, err := png.Decode(f)
			if err != nil {
				t.Fatal(err)
			}
			r, _, _, _ := img.At(0, 0).RGBA()
			want := uint32(100 * 257)
			if skip {
				want = 50 * 257
			}
			if r != want {
				t.Fatalf("output red = %d, want %d", r, want)
			}
			leftovers, err := filepath.Glob(filepath.Join(opts.outDir, "*.tmp"))
			if err != nil || len(leftovers) > 0 {
				t.Fatalf("temporary files: %v, %v", leftovers, err)
			}
		})
	}
}
