package main

import (
	"bufio"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

type imageResult struct {
	filePath, customBase, destPath, name, hiragana, katakana, hepburn string
	hasPronunciation                                                  bool
	rawAliases, warnings                                              []string
	err                                                               error
}

func progressLine(done, total int, label string) string {
	percent := done * 100 / total
	filled := percent * 10 / 100
	line := fmt.Sprintf("(%s%s) %3d%% (%d/%d)", strings.Repeat("=", filled), strings.Repeat("-", 10-filled), percent, done, total)
	if label != "" {
		line = label + " ………… " + line
	}
	return line
}

// Resolve interactive names before workers start, and retain input order for ZIP metadata.
func runImageBatches(files []string, opts *appOptions, reader *bufio.Reader) []imageResult {
	results := make([]imageResult, len(files))
	var groups [][]int
	directories := make(map[string]int)
	for i, file := range files {
		r := &results[i]
		r.filePath = file
		r.customBase, r.name, r.hiragana, r.katakana, r.hepburn, r.hasPronunciation, r.rawAliases = computeEmojiName(file, opts.zipMode, opts.namePrefix, opts.nameSuffix, reader, opts.filenameOption)
		key := ""
		if opts.recursive {
			key, _ = filepath.Abs(filepath.Dir(file))
		}
		group, ok := directories[key]
		if !ok {
			group = len(groups)
			directories[key] = group
			groups = append(groups, nil)
		}
		groups[group] = append(groups[group], i)
	}
	for _, indices := range groups {
		label := filepath.Clean(filepath.Dir(files[indices[0]]))
		for _, i := range indices {
			if filepath.Clean(filepath.Dir(files[i])) != label {
				label = "Total"
				break
			}
		}
		runImageGroup(indices, results, opts, label)
	}
	return results
}

// Files with the same output name share a job so overwrites and -skip retain input order.
func runImageGroup(indices []int, results []imageResult, opts *appOptions, label string) {
	var batches [][]int
	byDestination := make(map[string]int)
	for _, i := range indices {
		r := &results[i]
		out := opts.outDir
		if out == "" {
			out = filepath.Join(filepath.Dir(r.filePath), "output")
		}
		base := r.customBase
		ext := filepath.Ext(r.filePath)
		if base == "" {
			base = strings.TrimSuffix(filepath.Base(r.filePath), ext)
		}
		dest, _ := filepath.Abs(filepath.Join(out, base+opts.suffix+ext))
		src, _ := filepath.Abs(r.filePath)
		if dest == src {
			dest, _ = filepath.Abs(filepath.Join(out, base+opts.suffix+"_resized"+ext))
		}
		if runtime.GOOS == "windows" {
			dest = strings.ToLower(dest)
		}
		batch, ok := byDestination[dest]
		if !ok {
			batch = len(batches)
			byDestination[dest] = batch
			batches = append(batches, nil)
		}
		batches[batch] = append(batches[batch], i)
	}
	workers := opts.jobs
	if workers < 1 {
		workers = runtime.GOMAXPROCS(0)
	}
	if workers > len(batches) {
		workers = len(batches)
	}
	jobs := make(chan []int, len(batches))
	completed := make(chan int)
	for _, batch := range batches {
		jobs <- batch
	}
	close(jobs)
	fmt.Print(progressLine(0, len(indices), label))
	for worker := 0; worker < workers; worker++ {
		go func() {
			for batch := range jobs {
				for _, i := range batch {
					r := &results[i]
					rect, autoRect, trim, opaque := opts.rect, opts.autoRect.Active, opts.trim, opts.trimOpaque
					if opts.filenameOption {
						option := parseFilenameOption(r.filePath)
						if option.HasR {
							rect, autoRect = true, false
						}
						if option.HasS {
							rect, autoRect = false, false
						}
						trim = trim || option.HasT
						opaque = opaque || option.HasO
					}
					r.destPath, _, r.err = processImage(r.filePath, opts.outDir, opts.size, opts.suffix, opts.noResize, rect, r.customBase, autoRect, opts.autoRect.Ratio, opts.skip, opts.noResizeIfSmall, trim, opaque, func(message string) { r.warnings = append(r.warnings, message) })
					completed <- i
				}
			}
		}()
	}
	for done := 1; done <= len(indices); done++ {
		i := <-completed
		r := &results[i]
		diagnostic := r.err != nil || len(r.warnings) > 0
		if diagnostic {
			fmt.Println()
			for _, warning := range r.warnings {
				fmt.Printf("Warning: %s: %s\n", r.filePath, warning)
			}
			if r.err != nil {
				fmt.Printf("Failed: %s: %v\n", r.filePath, r.err)
			}
		}
		fmt.Print("\r" + progressLine(done, len(indices), label))
	}
	fmt.Println()
}
