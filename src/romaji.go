package main

import (
	"strings"
)

var romajiMap = map[string]struct{ Kunrei, Hepburn string }{
	"きゃ": {"kya", "kya"}, "きゅ": {"kyu", "kyu"}, "きょ": {"kyo", "kyo"},
	"ぎゃ": {"gya", "gya"}, "ぎゅ": {"gyu", "gyu"}, "ぎょ": {"gyo", "gyo"},
	"しゃ": {"sya", "sha"}, "しゅ": {"syu", "shu"}, "しょ": {"syo", "sho"},
	"じゃ": {"zya", "ja"},  "じゅ": {"zyu", "ju"},  "じょ": {"zyo", "jo"},
	"ちゃ": {"tya", "cha"}, "ちゅ": {"tyu", "chu"}, "ちょ": {"tyo", "cho"},
	"ぢゃ": {"zya", "ja"},  "ぢゅ": {"zyu", "ju"},  "ぢょ": {"zyo", "jo"},
	"にゃ": {"nya", "nya"}, "にゅ": {"nyu", "nyu"}, "にょ": {"nyo", "nyo"},
	"ひゃ": {"hya", "hya"}, "ひゅ": {"hyu", "hyu"}, "ひょ": {"hyo", "hyo"},
	"びゃ": {"bya", "bya"}, "びゅ": {"byu", "byu"}, "びょ": {"byo", "byo"},
	"ぴゃ": {"pya", "pya"}, "ぴゅ": {"pyu", "pyu"}, "ぴょ": {"pyo", "pyo"},
	"みゃ": {"mya", "mya"}, "みゅ": {"myu", "myu"}, "みょ": {"myo", "myo"},
	"りゃ": {"rya", "rya"}, "りゅ": {"ryu", "ryu"}, "りょ": {"ryo", "ryo"},
	"ふぁ": {"fa", "fa"},   "ふぃ": {"fi", "fi"},   "ふぇ": {"fe", "fe"},   "ふぉ": {"fo", "fo"},
	"ゔぁ": {"va", "va"},   "ゔぃ": {"vi", "vi"},   "ゔぇ": {"ve", "ve"},   "ゔぉ": {"vo", "vo"},
	"ちぇ": {"tye", "che"}, "しぇ": {"sye", "she"}, "じぇ": {"zye", "je"},
	"てぃ": {"thi", "thi"}, "でぃ": {"dhi", "dhi"},
	"てゅ": {"thu", "thu"}, "でゅ": {"dhu", "dhu"},
	"とぅ": {"twu", "twu"}, "どぅ": {"dwu", "dwu"},
	"つぁ": {"tsa", "tsa"}, "つぃ": {"tsi", "tsi"}, "つぇ": {"tse", "tse"}, "つぉ": {"tso", "tso"},
	"うぃ": {"wi", "wi"}, "うぇ": {"we", "we"}, "うぉ": {"wo", "wo"},
	"いぇ": {"ye", "ye"},
	"ふゃ": {"fya", "fya"}, "ふゅ": {"fyu", "fyu"}, "ふょ": {"fyo", "fyo"},
	"ゔゃ": {"vya", "vya"}, "ゔゅ": {"vyu", "vyu"}, "ゔょ": {"vyo", "vyo"},
	"くぁ": {"kwa", "kwa"}, "くぃ": {"kwi", "kwi"}, "くぇ": {"kwe", "kwe"}, "くぉ": {"kwo", "kwo"},
	"ぐぁ": {"gwa", "gwa"}, "ぐぃ": {"gwi", "gwi"}, "ぐぇ": {"gwe", "gwe"}, "ぐぉ": {"gwo", "gwo"},

	"あ": {"a", "a"}, "い": {"i", "i"}, "う": {"u", "u"}, "え": {"e", "e"}, "お": {"o", "o"},
	"か": {"ka", "ka"}, "き": {"ki", "ki"}, "く": {"ku", "ku"}, "け": {"ke", "ke"}, "こ": {"ko", "ko"},
	"が": {"ga", "ga"}, "ぎ": {"gi", "gi"}, "ぐ": {"gu", "gu"}, "げ": {"ge", "ge"}, "ご": {"go", "go"},
	"さ": {"sa", "sa"}, "し": {"si", "shi"}, "す": {"su", "su"}, "せ": {"se", "se"}, "そ": {"so", "so"},
	"ざ": {"za", "za"}, "じ": {"zi", "ji"}, "ず": {"zu", "zu"}, "ぜ": {"ze", "ze"}, "ぞ": {"zo", "zo"},
	"た": {"ta", "ta"}, "ち": {"ti", "chi"}, "つ": {"tu", "tsu"}, "て": {"te", "te"}, "と": {"to", "to"},
	"だ": {"da", "da"}, "ぢ": {"zi", "ji"}, "づ": {"zu", "zu"}, "で": {"de", "de"}, "ど": {"do", "do"},
	"な": {"na", "na"}, "に": {"ni", "ni"}, "ぬ": {"nu", "nu"}, "ね": {"ne", "ne"}, "の": {"no", "no"},
	"は": {"ha", "ha"}, "ひ": {"hi", "hi"}, "ふ": {"hu", "fu"}, "へ": {"he", "he"}, "ほ": {"ho", "ho"},
	"ば": {"ba", "ba"}, "び": {"bi", "bi"}, "ぶ": {"bu", "bu"}, "べ": {"be", "be"}, "ぼ": {"bo", "bo"},
	"ぱ": {"pa", "pa"}, "ぴ": {"pi", "pi"}, "ぷ": {"pu", "pu"}, "ぺ": {"pe", "pe"}, "ぽ": {"po", "po"},
	"ま": {"ma", "ma"}, "み": {"mi", "mi"}, "む": {"mu", "mu"}, "め": {"me", "me"}, "も": {"mo", "mo"},
	"や": {"ya", "ya"}, "ゆ": {"yu", "yu"}, "よ": {"yo", "yo"},
	"ら": {"ra", "ra"}, "り": {"ri", "ri"}, "る": {"ru", "ru"}, "れ": {"re", "re"}, "ろ": {"ro", "ro"},
	"わ": {"wa", "wa"}, "を": {"o", "wo"}, "ん": {"n", "n"},
	"ゔ": {"vu", "vu"},
	"ゐ": {"i", "i"}, "ゑ": {"e", "e"},
	"ぁ": {"la", "la"}, "ぃ": {"li", "li"}, "ぅ": {"lu", "lu"}, "ぇ": {"le", "le"}, "ぉ": {"lo", "lo"},
	"ゃ": {"lya", "lya"}, "ゅ": {"lyu", "lyu"}, "ょ": {"lyo", "lyo"},
	"っ": {"ltu", "ltu"},
	"ゎ": {"lwa", "lwa"},
	"ゕ": {"lka", "lka"}, "ゖ": {"lke", "lke"},
}

func isYoonSuffix(r rune) bool {
	switch r {
	case 'ゃ', 'ゅ', 'ょ', 'ぁ', 'ぃ', 'ぅ', 'ぇ', 'ぉ':
		return true
	}
	return false
}

func isDoubleableConsonant(b byte) bool {
	switch b {
	case 'k', 's', 't', 'p', 'b', 'c', 'd', 'f', 'g', 'j', 'm', 'r', 'v', 'z':
		return true
	}
	return false
}

func katakanaToHiragana(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r >= 0x30A1 && r <= 0x30F6 {
			sb.WriteRune(r - 0x60)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func hiraganaToKatakana(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r >= 0x3041 && r <= 0x3096 {
			sb.WriteRune(r + 0x60)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func containsJapanese(s string) bool {
	for _, r := range s {
		if (r >= 0x3040 && r <= 0x309F) || (r >= 0x30A0 && r <= 0x30FF) || (r >= 0x4E00 && r <= 0x9FFF) {
			return true
		}
	}
	return false
}

func hiraganaToRomaji(input string) (kunrei string, hepburn string) {
	normalized := katakanaToHiragana(input)
	var kResult strings.Builder
	var hResult strings.Builder

	runes := []rune(normalized)
	i := 0
	n := len(runes)

	for i < n {
		if runes[i] == 'っ' {
			doubled := false
			if i+1 < n {
				nextRune := runes[i+1]
				nextStr := string(nextRune)
				if i+2 < n && isYoonSuffix(runes[i+2]) {
					nextStr = string(runes[i+1 : i+3])
				}

				kNext, hNext := lookupRomaji(nextStr)

				kConsonant := len(kNext) > 0 && isDoubleableConsonant(kNext[0])
				hConsonant := len(hNext) > 0 && isDoubleableConsonant(hNext[0])

				if kConsonant {
					kResult.WriteByte(kNext[0])
				}
				if hConsonant {
					if strings.HasPrefix(hNext, "ch") {
						hResult.WriteByte('t')
					} else {
						hResult.WriteByte(hNext[0])
					}
				}
				if kConsonant || hConsonant {
					doubled = true
				}
			}
			if !doubled {
				kResult.WriteString("ltu")
				hResult.WriteString("ltu")
			}
			i++
			continue
		}

		if runes[i] == 'ー' {
			i++
			continue
		}

		if i+1 < n {
			twoChars := string(runes[i : i+2])
			if val, ok := romajiMap[twoChars]; ok {
				kResult.WriteString(val.Kunrei)
				hResult.WriteString(val.Hepburn)
				i += 2
				continue
			}
		}

		charStr := string(runes[i])
		if val, ok := romajiMap[charStr]; ok {
			kResult.WriteString(val.Kunrei)
			hResult.WriteString(val.Hepburn)
		} else {
			kResult.WriteRune(runes[i])
			hResult.WriteRune(runes[i])
		}
		i++
	}

	return kResult.String(), hResult.String()
}

func lookupRomaji(s string) (string, string) {
	if val, ok := romajiMap[s]; ok {
		return val.Kunrei, val.Hepburn
	}
	return "", ""
}

func isPureHiraganaOrSafe(s string) bool {
	hasHiragana := false
	for _, r := range s {
		if (r >= 0x3041 && r <= 0x3096) || r == 0x3094 || r == 'ー' {
			hasHiragana = true
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return hasHiragana
}

func sanitizeFileName(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('_')
		}
	}
	res := sb.String()
	for strings.Contains(res, "__") {
		res = strings.ReplaceAll(res, "__", "_")
	}
	return strings.Trim(res, "_")
}
