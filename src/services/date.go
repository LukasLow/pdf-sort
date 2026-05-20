package services

import (
	"fmt"
	"regexp"
)

func FindDate(text string) (y, m, d string) {
	if text == "" {
		return "", "", ""
	}

	reSimple := regexp.MustCompile(`\b(\d{1,2})\.(\d{1,2})\.(\d{4})\b`)
	if match := reSimple.FindStringSubmatch(text); match != nil {
		return match[3], fmt.Sprintf("%02s", match[2]), fmt.Sprintf("%02s", match[1])
	}

	reISO := regexp.MustCompile(`\b(\d{4})-(\d{2})-(\d{2})\b`)
	if match := reISO.FindStringSubmatch(text); match != nil {
		return match[1], match[2], match[3]
	}

	return "", "", ""
}
