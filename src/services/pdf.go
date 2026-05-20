package services

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func ExtractText(path string) string {
	cmd := exec.Command("pdftotext", "-l", "1", path, "-")

	var out bytes.Buffer
	cmd.Stdout = &out

	_ = cmd.Run()

	return out.String()
}

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

func HumanFileSize(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "?"
	}

	size := float64(info.Size())

	if size < 1024 {
		return fmt.Sprintf("%.0f B", size)
	}

	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", size/1024)
	}

	return fmt.Sprintf("%.1f MB", size/(1024*1024))
}

func EstimatePPI(path string) string {
	cmd := exec.Command(
		"pdfinfo",
		path,
	)

	out, err := cmd.Output()
	if err != nil {
		return "Unbekannt"
	}

	txt := string(out)

	if strings.Contains(txt, "A4") {
		return "~300 PPI"
	}

	return "Unbekannt"
}
