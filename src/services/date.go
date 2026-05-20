package services

import (
	"fmt"
	"regexp"
	"strings"
)

func FindDate(text string) (y, m, d string) {
	if text == "" {
		return "", "", ""
	}

	// numeric date with dot separator (DD.MM.YYYY)
	reSimple := regexp.MustCompile(`\b(\d{1,2})\.(\d{1,2})\.(\d{4})\b`)
	if match := reSimple.FindStringSubmatch(text); match != nil {
		return match[3], fmt.Sprintf("%02s", match[2]), fmt.Sprintf("%02s", match[1])
	}

	// ISO format YYYY-MM-DD
	reISO := regexp.MustCompile(`\b(\d{4})-(\d{2})-(\d{2})\b`)
	if match := reISO.FindStringSubmatch(text); match != nil {
		return match[1], match[2], match[3]
	}

	// month name handling (German and English, full and abbreviated)
	monthMap := map[string]string{
		// German full names and abbreviations
		"januar": "01", "jan": "01",
		"februar": "02", "feb": "02",
		"märz": "03", "maerz": "03", "mrz": "03",
		"april": "04", "apr": "04",
		"mai":  "05",
		"juni": "06", "jun": "06",
		"juli": "07", "jul": "07",
		"august": "08", "aug": "08",
		"september": "09", "sep": "09",
		"oktober": "10", "okt": "10",
		"november": "11", "nov": "11",
		"dezember": "12", "dez": "12",
		// English full names and unique abbreviations
		"january":  "01",
		"february": "02",
		"march":    "03", "mar": "03",
		"may":     "05",
		"october": "10", "oct": "10",
		"december": "12", "dec": "12",
	}

	// Patterns: "DD Month YYYY" or "YYYY Month DD"
	reMonth := regexp.MustCompile(`(?i)\b(\d{1,2})\s+([a-zäöü]+)\s+(\d{4})\b`)
	if match := reMonth.FindStringSubmatch(text); match != nil {
		month := strings.ToLower(match[2])
		if mNum, ok := monthMap[month]; ok {
			return match[3], mNum, fmt.Sprintf("%02s", match[1])
		}
	}
	reMonthAlt := regexp.MustCompile(`(?i)\b(\d{4})\s+([a-zäöü]+)\s+(\d{1,2})\b`)
	if match := reMonthAlt.FindStringSubmatch(text); match != nil {
		month := strings.ToLower(match[2])
		if mNum, ok := monthMap[month]; ok {
			return match[1], mNum, fmt.Sprintf("%02s", match[3])
		}
	}

	return "", "", ""
}
