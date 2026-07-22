package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

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

type Config struct {
	Size            *int        `json:"size"`
	OutDir          *string     `json:"out"`
	Suffix          *string     `json:"suffix"`
	NamePrefix      *string     `json:"name_prefix"`
	NameSuffix      *string     `json:"name_suffix"`
	Recursive       *bool       `json:"recursive"`
	NoResize        *bool       `json:"no_resize"`
	NoResizeIfSmall *bool       `json:"no_resize_if_small"`
	Rect            *bool       `json:"rect"`
	ZipMode         *bool       `json:"zip"`
	AutoRect        interface{} `json:"auto_rect"`
	Skip            *bool       `json:"skip"`
	Category        *string     `json:"category"`
	License         *string     `json:"license"`
	FilenameOption  *bool       `json:"filename_option"`
}

func parseAndApplyConfig(configPath string, seenFlags map[string]bool,
	size *int, outDir *string, suffix *string, namePrefix *string, nameSuffix *string,
	recursive *bool, noResize *bool, rect *bool, zipMode *bool, skip *bool,
	autoRect *AutoRectValue, cfgCategory *string, cfgLicense *string, noResizeIfSmall *bool,
	filenameOption *bool) error {

	var shouldLoadConfig bool
	var finalConfigPath string

	if configPath != "" {
		shouldLoadConfig = true
		finalConfigPath = configPath
	}

	if !shouldLoadConfig {
		return nil
	}

	cfgFile, err := os.Open(finalConfigPath)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer cfgFile.Close()

	var cfg Config
	decoder := json.NewDecoder(cfgFile)
	if err := decoder.Decode(&cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.Size != nil && !seenFlags["size"] {
		*size = *cfg.Size
	}
	if cfg.OutDir != nil && !seenFlags["out"] {
		*outDir = *cfg.OutDir
	}
	if cfg.Suffix != nil && !seenFlags["suffix"] {
		*suffix = *cfg.Suffix
	}
	if cfg.NamePrefix != nil && !seenFlags["name-prefix"] {
		*namePrefix = *cfg.NamePrefix
	}
	if cfg.NameSuffix != nil && !seenFlags["name-suffix"] {
		*nameSuffix = *cfg.NameSuffix
	}
	if cfg.Recursive != nil && !seenFlags["recursive"] {
		*recursive = *cfg.Recursive
	}
	if cfg.NoResize != nil && !seenFlags["no-resize"] {
		*noResize = *cfg.NoResize
	}
	if cfg.NoResizeIfSmall != nil && !seenFlags["no-resize-if-small"] {
		*noResizeIfSmall = *cfg.NoResizeIfSmall
	}
	if cfg.Rect != nil && !seenFlags["rect"] {
		*rect = *cfg.Rect
	}
	if cfg.ZipMode != nil && !seenFlags["zip"] {
		*zipMode = *cfg.ZipMode
	}
	if cfg.Skip != nil && !seenFlags["skip"] {
		*skip = *cfg.Skip
	}
	if cfg.FilenameOption != nil && !seenFlags["filename-option"] {
		*filenameOption = *cfg.FilenameOption
	}
	if cfg.Category != nil {
		*cfgCategory = *cfg.Category
	}
	if cfg.License != nil {
		*cfgLicense = *cfg.License
	}
	if cfg.AutoRect != nil && !seenFlags["auto-rect"] {
		switch v := cfg.AutoRect.(type) {
		case bool:
			autoRect.Active = v
			autoRect.Ratio = 0
		case float64:
			autoRect.Active = true
			autoRect.Ratio = v
		case string:
			if err := autoRect.Set(v); err != nil {
				return fmt.Errorf("invalid auto_rect in config: %w", err)
			}
		default:
			return fmt.Errorf("invalid type for auto_rect in config")
		}
	}

	return nil
}
