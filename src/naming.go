package main

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strings"
)

func computeEmojiName(filePath string, zipMode bool, namePrefix string, nameSuffix string, reader *bufio.Reader) (customBase string, name string, hiragana string, katakana string, hepburn string, hasPronunciation bool, rawAliases []string) {
	ext := filepath.Ext(filePath)
	rawBase := strings.TrimSuffix(filepath.Base(filePath), ext)

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
