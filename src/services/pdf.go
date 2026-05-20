package services

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func ExtractText(path string) string {
	cmd := exec.Command("pdftotext", "-l", "1", path, "-")

	var out bytes.Buffer
	cmd.Stdout = &out

	_ = cmd.Run()

	return out.String()
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
