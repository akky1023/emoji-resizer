package main

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"

	// pure Go WebP library supporting lossless encoding
	"github.com/deepteams/webp"
)

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

func processImage(srcPath string, outDir string, targetSize int, suffix string, noResize bool, rect bool, customBase string, autoRectActive bool, autoRectRatio float64, skipExist bool, noResizeIfSmall bool) (string, bool, error) {
	// 1. Open the source file
	file, err := os.Open(srcPath)
	if err != nil {
		return "", false, fmt.Errorf("failed to open file: %w", err)
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
				return "", false, fmt.Errorf("failed to seek/decode GIF: %w", errSeek)
			}
		}
	}

	if img == nil {
		img, format, err = image.Decode(file)
		if err != nil {
			return "", false, fmt.Errorf("failed to decode image: %w", err)
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
		threshold := 1.618
		if autoRectRatio > 1.0 {
			threshold = autoRectRatio
		}
		actualRect = ratio > threshold
	}

	if noResizeIfSmall && w > 0 && h > 0 {
		if actualRect {
			minDim := w
			if h < w {
				minDim = h
			}
			if minDim < targetSize {
				noResize = true
			}
		} else {
			maxDim := w
			if h > w {
				maxDim = h
			}
			if maxDim <= targetSize {
				noResize = true
			}
		}
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
		return "", false, fmt.Errorf("failed to create output directory: %w", err)
	}

	outFileName := base + suffix + ext
	destPath := filepath.Join(finalOutDir, outFileName)

	// Prevent overwriting the original file in any case
	absSrc, err := filepath.Abs(srcPath)
	if err != nil {
		return "", false, fmt.Errorf("failed to resolve absolute path for source file: %w", err)
	}
	absDest, err := filepath.Abs(destPath)
	if err != nil {
		return "", false, fmt.Errorf("failed to resolve absolute path for destination file: %w", err)
	}

	if absSrc == absDest {
		// If they are exactly the same, force a suffix to be safe
		outFileName = base + suffix + "_resized" + ext
		destPath = filepath.Join(finalOutDir, outFileName)
		absDest, err = filepath.Abs(destPath)
		if err != nil {
			return "", false, fmt.Errorf("failed to resolve absolute path for safe destination file: %w", err)
		}
	}

	if absSrc == absDest {
		return "", false, fmt.Errorf("refusing to write: output file path is identical to source file path")
	}

	if skipExist {
		if _, err := os.Stat(destPath); err == nil {
			return destPath, true, nil
		}
	}

	// Write to a temporary file first, then rename it (atomic swap) to prevent corruption
	tmpPath := destPath + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return "", false, fmt.Errorf("failed to create temporary file: %w", err)
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
		return "", false, fmt.Errorf("failed to encode/save image: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return "", false, fmt.Errorf("failed to close temporary file: %w", err)
	}

	if err := safeRenameOrCopy(tmpPath, destPath); err != nil {
		return "", false, fmt.Errorf("failed to rename temporary file to destination: %w", err)
	}

	return destPath, false, nil
}

// safeRenameOrCopy moves a file from src to dest. It handles cases where dest already exists
// (which can fail on Windows via os.Rename) and cases where src and dest are on different
// filesystems/devices (which fails with EXDEV).
func safeRenameOrCopy(src, dest string) error {
	if src == dest {
		return nil
	}

	// For Windows: if the destination file exists, remove it first
	// to prevent os.Rename from failing.
	if _, err := os.Stat(dest); err == nil {
		if err := os.Remove(dest); err != nil {
			return fmt.Errorf("failed to remove existing file: %w", err)
		}
	}

	// Try standard rename first
	err := os.Rename(src, dest)
	if err == nil {
		return nil
	}

	// Fallback to copy if rename failed (e.g. cross-device link EXDEV)
	srcFile, openErr := os.Open(src)
	if openErr != nil {
		return fmt.Errorf("rename failed: %w; fallback open failed: %v", err, openErr)
	}
	defer srcFile.Close()

	destFile, createErr := os.Create(dest)
	if createErr != nil {
		return fmt.Errorf("rename failed: %w; fallback create failed: %v", err, createErr)
	}
	defer destFile.Close()

	if _, copyErr := io.Copy(destFile, srcFile); copyErr != nil {
		return fmt.Errorf("rename failed: %w; fallback copy failed: %v", err, copyErr)
	}

	srcFile.Close() // close before removal
	os.Remove(src)
	return nil
}

