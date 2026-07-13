package main

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strings"
)

func computeEmojiName(filePath string, zipMode bool, namePrefix string, nameSuffix string, reader *bufio.Reader) (customBase string, name string, hiragana string, katakana string, hepburn string, hasPronunciation bool) {
	ext := filepath.Ext(filePath)
	base := strings.TrimSuffix(filepath.Base(filePath), ext)

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
		customBase = namePrefix + base + nameSuffix
	}
	return customBase, name, hiragana, katakana, hepburn, hasPronunciation
}
