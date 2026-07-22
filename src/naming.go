package main

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strings"
)

type FilenameOptionResult struct {
	HasR                bool
	HasS                bool
	HasInvalidOptionPos bool
	RawOpt              string
	CleanRawBase        string
}

func isValidOptionString(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, ch := range s {
		lower := strings.ToLower(string(ch))
		if lower != "r" && lower != "s" {
			return false
		}
	}
	return true
}

func parseFilenameOption(filePath string) FilenameOptionResult {
	ext := filepath.Ext(filePath)
	rawBase := strings.TrimSuffix(filepath.Base(filePath), ext)

	res := FilenameOptionResult{
		CleanRawBase: rawBase,
	}

	work := rawBase
	for {
		idx := strings.LastIndex(work, ".")
		if idx == -1 || idx == len(work)-1 {
			break
		}
		optStr := work[idx+1:]
		if !isValidOptionString(optStr) {
			break
		}

		res.RawOpt = optStr + res.RawOpt
		optLower := strings.ToLower(optStr)
		if strings.Contains(optLower, "r") {
			res.HasR = true
		}
		if strings.Contains(optLower, "s") {
			res.HasS = true
		}

		work = work[:idx]
	}

	res.CleanRawBase = work

	parts := strings.Split(work, "@")
	for _, part := range parts {
		idx := strings.LastIndex(part, ".")
		if idx != -1 && idx < len(part)-1 {
			optStr := part[idx+1:]
			if isValidOptionString(optStr) {
				res.HasInvalidOptionPos = true
				break
			}
		}
	}

	return res
}

func computeEmojiName(filePath string, zipMode bool, namePrefix string, nameSuffix string, reader *bufio.Reader, filenameOption bool) (customBase string, name string, hiragana string, katakana string, hepburn string, hasPronunciation bool, rawAliases []string) {
	ext := filepath.Ext(filePath)
	rawBase := strings.TrimSuffix(filepath.Base(filePath), ext)

	if filenameOption {
		fnOpt := parseFilenameOption(filePath)
		if fnOpt.CleanRawBase != "" {
			rawBase = fnOpt.CleanRawBase
		}
	}

	parts := strings.Split(rawBase, "@")
	base := parts[0]
	if len(parts) > 1 {
		for _, a := range parts[1:] {
			trimmed := strings.TrimSpace(a)
			if trimmed != "" {
				rawAliases = append(rawAliases, trimmed)
			}
		}
	}

	if zipMode {
		if isPureHiraganaOrSafe(base) {
			hiragana = base
			katakana = hiraganaToKatakana(hiragana)
			var hepburnRaw string
			name, hepburnRaw = hiraganaToRomaji(hiragana)
			name = strings.ToLower(name)
			hepburn = strings.ToLower(hepburnRaw)
			hasPronunciation = true
		} else if containsJapanese(base) {
			fmt.Printf("ファイル名 '%s' のひらがな表記を入力してください (英語などの場合はそのままEnter): ", base)
			input, err := reader.ReadString('\n')
			if err == nil {
				hiragana = strings.TrimSpace(input)
			} else {
				hiragana = ""
			}
			if hiragana != "" {
				katakana = hiraganaToKatakana(hiragana)
				var hepburnRaw string
				name, hepburnRaw = hiraganaToRomaji(hiragana)
				name = strings.ToLower(name)
				hepburn = strings.ToLower(hepburnRaw)
				hasPronunciation = true
			} else {
				name = strings.ToLower(base)
			}
		} else {
			name = strings.ToLower(base)
		}
		customBase = namePrefix + name + nameSuffix
	} else {
		customBase = namePrefix + rawBase + nameSuffix
	}
	return customBase, name, hiragana, katakana, hepburn, hasPronunciation, rawAliases
}

func expandAlias(alias string) []string {
	var results []string
	results = addUnique(results, alias)

	hira := katakanaToHiragana(alias)
	kata := hiraganaToKatakana(alias)

	if isPureHiraganaOrSafe(hira) {
		results = addUnique(results, hira)
		results = addUnique(results, kata)
		kunrei, hepburn := hiraganaToRomaji(hira)
		if kunrei != "" {
			results = addUnique(results, strings.ToLower(kunrei))
		}
		if hepburn != "" {
			results = addUnique(results, strings.ToLower(hepburn))
		}
	}
	return results
}
