package main

import (
	"archive/zip"
	"bufio"
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
	for _, name := range []string{"foo.png", "animals~bar.png", "animals~faces~baz.png", "sub1/deep/qux.png", "カテゴリ~カテゴリ2~ねこ@cat.png"} {
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
		recurse  bool
	}{
		{"default", []string{"-category", "root"}, `{}`, "root", false},
		{"recursive", []string{"-recursive", "-category", "root"}, `{}`, "root", true},
		{"config", nil, `{"category":"configured"}`, "configured", false},
		{"category_override", []string{"-category", "root"}, `{"category":"configured"}`, "root", false},
		{"empty_category", []string{"-category="}, `{"category":"configured"}`, "", false},
		{"filename_option", []string{"-filename-option", "-category", "root"}, `{}`, "root", false},
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
			joinCategory := func(path string) string {
				return strings.TrimPrefix(tc.category+"/"+path, "/")
			}
			want := map[string]string{"foo": tc.category, "bar": joinCategory("animals"), "baz": joinCategory("animals/faces"), "neko": joinCategory("カテゴリ/カテゴリ2")}
			if tc.recurse {
				want["qux"] = tc.category
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
	duplicate := filepath.Join(tmp, "other~foo.png")
	data, err := os.ReadFile(filepath.Join(root, "foo.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(duplicate, data, 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary, "-check", "-zip", filepath.Join(root, "foo.png"), duplicate)
	output, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(output), duplicate) {
		t.Fatalf("category-free duplicate check: %v\n%s", err, output)
	}
}

func TestCategoryFilenameNaming(t *testing.T) {
	file := "カテゴリ~カテゴリ2~ねこ@cat~alias.r.png"
	path, category := splitCategoryFilename(file, "root")
	if path != "ねこ@cat~alias.r.png" || category != "root/カテゴリ/カテゴリ2" {
		t.Fatalf("split = %q, %q", path, category)
	}
	base, name, _, _, _, _, aliases := computeEmojiName(file, true, "pre_", "_post", bufio.NewReader(strings.NewReader("")), true)
	if base != "pre_neko_post" || name != "neko" || len(aliases) != 1 || aliases[0] != "cat~alias" {
		t.Fatalf("ZIP name = %q, %q, aliases = %v", base, name, aliases)
	}
	base, _, _, _, _, _, _ = computeEmojiName(file, false, "", "", nil, false)
	if base != strings.TrimSuffix(file, ".png") {
		t.Fatalf("non-ZIP name = %q", base)
	}
}
