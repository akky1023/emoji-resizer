package main

import (
	"archive/zip"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGroupingCLI(t *testing.T) {
	tmp := t.TempDir()
	binary := filepath.Join(tmp, "emoji-resizer.exe")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	root := filepath.Join(tmp, "root")
	for _, name := range []string{"foo.png", "sub1/bar.png", "sub2/baz.png", "sub1/deep/qux.png"} {
		filePath := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filePath)
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(f, image.NewRGBA(image.Rect(0, 0, 8, 8)))
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			t.Fatalf("write image: %v, %v", err, closeErr)
		}
	}

	for _, tc := range []struct {
		name     string
		flags    []string
		config   string
		category string
		grouped  bool
		recurse  bool
	}{
		{"grouping", []string{"-grouping", "-category", "maincat"}, `{}`, "maincat", true, true},
		{"recursive_false_after", []string{"-grouping", "-recursive=false", "-category", "maincat"}, `{}`, "maincat", true, true},
		{"recursive_false_before", []string{"-recursive=false", "-grouping", "-category", "maincat"}, `{}`, "maincat", true, true},
		{"config", []string{"-recursive=false"}, `{"grouping":true,"recursive":false,"category":"configured"}`, "configured", true, true},
		{"category_override", []string{"-grouping", "-category", "maincat"}, `{"category":"configured"}`, "maincat", true, true},
		{"empty_category", []string{"-grouping", "-category="}, `{"category":"configured"}`, "", true, true},
		{"disabled", []string{"-grouping=false", "-recursive=false", "-category", "maincat"}, `{"grouping":true}`, "maincat", false, false},
		{"recursive_only", []string{"-recursive", "-category", "maincat"}, `{}`, "maincat", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := filepath.Join(tmp, tc.name)
			configPath := filepath.Join(tmp, tc.name+".json")
			if err := os.WriteFile(configPath, []byte(tc.config), 0644); err != nil {
				t.Fatal(err)
			}
			args := []string{"-zip", "-config", configPath, "-out", out}
			args = append(args, tc.flags...)
			args = append(args, root)
			cmd := exec.Command(binary, args...)
			cmd.Stdin = strings.NewReader("\n\n")
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("run: %v\n%s", err, output)
			}
			want := map[string]string{"foo": tc.category}
			if tc.recurse {
				for name, dir := range map[string]string{"bar": "sub1", "baz": "sub2", "qux": "sub1/deep"} {
					category := tc.category
					if tc.grouped {
						category = strings.TrimPrefix(category+"/"+dir, "/")
					}
					want[name] = category
				}
			}
			archives, err := filepath.Glob(filepath.Join(out, "*.zip"))
			if err != nil || len(archives) == 0 {
				t.Fatalf("find ZIPs: %v, %v", archives, err)
			}
			seen := make(map[string]bool)
			for _, archive := range archives {
				zr, err := zip.OpenReader(archive)
				if err != nil {
					t.Fatal(err)
				}
				defer zr.Close()
				f, err := zr.Open("meta.json")
				if err != nil {
					t.Fatal(err)
				}
				var meta MisskeyMeta
				err = json.NewDecoder(f).Decode(&meta)
				f.Close()
				if err != nil {
					t.Fatal(err)
				}
				for _, entry := range meta.Emojis {
					expected, ok := want[entry.Emoji.Name]
					if !ok || entry.Emoji.Category != expected {
						t.Errorf("%s: %s category = %q, want %q", archive, entry.Emoji.Name, entry.Emoji.Category, expected)
					}
					seen[entry.Emoji.Name] = true
				}
				if tc.recurse && strings.HasSuffix(archive, "_all.zip") && len(meta.Emojis) != len(want) {
					t.Errorf("combined ZIP has %d emojis, want %d", len(meta.Emojis), len(want))
				}
			}
			if len(seen) != len(want) {
				t.Errorf("exported %v, want %v", seen, want)
			}
		})
	}
}
