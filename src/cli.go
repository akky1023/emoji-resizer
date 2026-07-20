package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type appOptions struct {
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
	cfgCategory     string
	cfgLicense      string
	category        string
	license         string
	absOutDir       string
}

// preprocessConfigArgs handles optional arguments for -config / --config.
func preprocessConfigArgs(args []string) []string {
	var newArgs []string
	for i := 0; i < len(args); i++ {
		newArgs = append(newArgs, args[i])
		if args[i] == "-config" || args[i] == "--config" {
			nextIsVal := false
			if i+1 < len(args) {
				if !strings.HasPrefix(args[i+1], "-") {
					nextIsVal = true
				}
			}
			if !nextIsVal {
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
	return newArgs
}

// parseOptions parses command-line flags and merges config options.
func parseOptions() (*appOptions, []string, error) {
	opts := &appOptions{}

	flag.IntVar(&opts.size, "size", 128, "target resize square size in pixels")
	flag.StringVar(&opts.outDir, "out", "", "custom output directory path (default: 'output' directory inside the source file's directory)")
	flag.StringVar(&opts.suffix, "suffix", "", "suffix to append to the output filename (e.g. '_resized')")
	flag.StringVar(&opts.namePrefix, "name-prefix", "", "prefix to prepend to the emoji name")
	flag.StringVar(&opts.nameSuffix, "name-suffix", "", "suffix to append to the emoji name")
	flag.StringVar(&opts.configPath, "config", "", "path to config file (default: './config.json' if exists)")
	flag.BoolVar(&opts.recursive, "r", false, "recursively scan directories")
	flag.BoolVar(&opts.noResize, "no-resize", false, "skip final resizing and keep the original square dimensions")
	flag.BoolVar(&opts.noResizeIfSmall, "no-resize-if-small", false, "skip resizing if the image is already smaller than the target size")
	flag.BoolVar(&opts.rect, "rect", false, "resize rectangle keeping aspect ratio, short side matches target size (no padding)")
	flag.BoolVar(&opts.showVersion, "version", false, "show version information and exit")
	flag.BoolVar(&opts.zipMode, "zip", false, "pack processed images into a Misskey-compatible emoji ZIP file")
	flag.Var(&opts.autoRect, "auto-rect", "automatically use rect mode if aspect ratio exceeds threshold (defaults to golden ratio ~1.618)")
	flag.BoolVar(&opts.skip, "skip", false, "skip resizing if the destination file already exists")
	flag.BoolVar(&opts.checkMode, "check", false, "check for duplicate emoji names after conversion")

	flag.Parse()

	seenFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		seenFlags[f.Name] = true
	})

	if err := parseAndApplyConfig(
		opts.configPath, seenFlags,
		&opts.size, &opts.outDir, &opts.suffix, &opts.namePrefix, &opts.nameSuffix,
		&opts.recursive, &opts.noResize, &opts.rect, &opts.zipMode, &opts.skip,
		&opts.autoRect, &opts.cfgCategory, &opts.cfgLicense, &opts.noResizeIfSmall,
	); err != nil {
		return nil, nil, err
	}

	if opts.size < 8 || opts.size > 8192 {
		return nil, nil, fmt.Errorf("invalid size %d (must be between 8 and 8192 px)", opts.size)
	}

	args := flag.Args()
	if len(args) == 0 {
		args = []string{"."}
	}

	if opts.outDir != "" {
		var err error
		opts.absOutDir, err = filepath.Abs(opts.outDir)
		if err != nil {
			fmt.Printf("Warning: failed to resolve absolute path for output directory: %v\n", err)
		}
	}

	return opts, args, nil
}

// promptCategoryAndLicense prompts the user for category and license inputs when in zip mode.
func promptCategoryAndLicense(reader *bufio.Reader, cfgCategory, cfgLicense string) (string, string) {
	var category, license string

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

	return category, license
}
