package main

import (
	"flag"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"

	// pure Go WebP library supporting lossless encoding
	"github.com/deepteams/webp"
)

var version = "devel"

func main() {
	var (
		size        int
		outDir      string
		suffix      string
		recursive   bool
		noResize    bool
		showVersion bool
	)

	flag.IntVar(&size, "size", 128, "target resize square size in pixels")
	flag.StringVar(&outDir, "out", "", "custom output directory path (default: 'output' directory inside the source file's directory)")
	flag.StringVar(&suffix, "suffix", "", "suffix to append to the output filename (e.g. '_resized')")
	flag.BoolVar(&recursive, "r", false, "recursively scan directories")
	flag.BoolVar(&noResize, "no-resize", false, "skip final resizing and keep the original square dimensions")
	flag.BoolVar(&showVersion, "version", false, "show version information and exit")
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
		fmt.Printf("Found %d image files. Starting processing (no-resize: padding only)...\n", len(filesToProcess))
	} else {
		fmt.Printf("Found %d image files. Starting processing (target size: %dx%d px)...\n", len(filesToProcess), size, size)
	}

	successCount := 0
	failureCount := 0
	for _, filePath := range filesToProcess {
		// Use backslashes on Windows for clean output log paths
		displayPath := filepath.Clean(filePath)
		fmt.Printf("Processing %s ... ", displayPath)
		err := processImage(filePath, outDir, size, suffix, noResize)
		if err != nil {
			fmt.Printf("Failed: %v\n", err)
			failureCount++
		} else {
			fmt.Println("Success")
			successCount++
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

func processImage(srcPath string, outDir string, targetSize int, suffix string, noResize bool) error {
	// 1. Open the source file
	file, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
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
				return fmt.Errorf("failed to seek/decode GIF: %w", errSeek)
			}
		}
	}

	if img == nil {
		img, format, err = image.Decode(file)
		if err != nil {
			return fmt.Errorf("failed to decode image: %w", err)
		}
	}

	// 3. Make square (padding)
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
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
	var finalImg image.Image
	if noResize {
		finalImg = canvas
	} else {
		resized := image.NewRGBA(image.Rect(0, 0, targetSize, targetSize))
		draw.CatmullRom.Scale(resized, resized.Bounds(), canvas, canvas.Bounds(), draw.Src, nil)
		finalImg = resized
	}

	// 5. Determine the output path
	base := strings.TrimSuffix(filepath.Base(srcPath), ext)

	// Generate output directory
	finalOutDir := outDir
	if finalOutDir == "" {
		finalOutDir = filepath.Join(filepath.Dir(srcPath), "output")
	}

	if err := os.MkdirAll(finalOutDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outFileName := base + suffix + ext
	destPath := filepath.Join(finalOutDir, outFileName)

	// Prevent overwriting the original file in any case
	absSrc, err := filepath.Abs(srcPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for source file: %w", err)
	}
	absDest, err := filepath.Abs(destPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for destination file: %w", err)
	}

	if absSrc == absDest {
		// If they are exactly the same, force a suffix to be safe
		outFileName = base + suffix + "_resized" + ext
		destPath = filepath.Join(finalOutDir, outFileName)
		absDest, err = filepath.Abs(destPath)
		if err != nil {
			return fmt.Errorf("failed to resolve absolute path for safe destination file: %w", err)
		}
	}

	if absSrc == absDest {
		return fmt.Errorf("refusing to write: output file path is identical to source file path")
	}

	// Write to a temporary file first, then rename it (atomic swap) to prevent corruption
	tmpPath := destPath + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
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
		return fmt.Errorf("failed to encode/save image: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("failed to rename temporary file to destination: %w", err)
	}

	return nil
}
