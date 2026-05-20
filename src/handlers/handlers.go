package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"pdf-sort/src/config"
	"pdf-sort/src/models"
	"pdf-sort/src/services"
	"pdf-sort/src/templates"
	"pdf-sort/src/utils"
)

func GetNextPDF(w http.ResponseWriter, r *http.Request) {
	files, _ := filepath.Glob(filepath.Join(config.InputDir, "*.pdf"))

	if len(files) == 0 {
		json.NewEncoder(w).Encode(models.NextResponse{})
		return
	}

	sort.Strings(files)

	targetFile := files[0]
	filename := filepath.Base(targetFile)

	text := services.ExtractText(targetFile)

	y, m, d := services.FindDate(text)

	cfg := services.LoadConfig()

	uiConfig := make(map[string][]string)

	for k, v := range cfg {
		uiConfig[k] = v.Info
	}

	suggestedCorr := ""
	suggestedInfo := ""

	for corr, val := range cfg {
		if strings.Contains(strings.ToLower(text), strings.ToLower(corr)) {
			suggestedCorr = corr

			if len(val.Info) > 0 {
				suggestedInfo = val.Info[0]
			}

			break
		}
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(models.NextResponse{
		Filename:      filename,
		Year:          y,
		Month:         m,
		Day:           d,
		SuggestedCorr: suggestedCorr,
		SuggestedInfo: suggestedInfo,
		ConfigData:    uiConfig,
		FileSize:      services.HumanFileSize(targetFile),
		PPI:           services.EstimatePPI(targetFile),
	})
}

func AddNewEntry(w http.ResponseWriter, r *http.Request) {
	var req models.NewEntryRequest

	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.Correspondent == "" {
		http.Error(w, "Name fehlt", http.StatusBadRequest)
		return
	}

	cfg := services.LoadConfig()

	val, ok := cfg[req.Correspondent]

	if !ok {
		val.Path = filepath.Join(config.BaseArch, req.Correspondent)
		val.Info = []string{}
	}

	if req.Info != "" {
		exists := false

		for _, info := range val.Info {
			if info == req.Info {
				exists = true
				break
			}
		}

		if !exists {
			val.Info = append(val.Info, req.Info)
		}
	}

	cfg[req.Correspondent] = val

	services.SaveConfig(cfg)

	w.WriteHeader(http.StatusOK)
}

func ProcessPDF(w http.ResponseWriter, r *http.Request) {
	var req models.ProcessRequest

	_ = json.NewDecoder(r.Body).Decode(&req)

	srcPath := filepath.Join(config.InputDir, req.Filename)

	cfg := services.LoadConfig()

	val, ok := cfg[req.Correspondent]

	if !ok {
		http.Error(w, "Korrespondent existiert nicht", http.StatusBadRequest)
		return
	}

	finalDestDir := filepath.Join(val.Path, req.Info)

	_ = os.MkdirAll(finalDestDir, 0755)

	finalName := req.Year + "-" + req.Month + "-" + req.Day +
		"_" + req.Correspondent +
		"_" + req.Info

	if req.Extra != "" {
		finalName += "_" + req.Extra
	}

	finalName += ".pdf"

	destPath := filepath.Join(finalDestDir, finalName)

	tempOcr := filepath.Join(config.InputDir, "ocr_"+finalName)

	text := services.ExtractText(srcPath)

	workingPath := srcPath

	if len(strings.TrimSpace(text)) == 0 {
		cmd := exec.Command(
			"ocrmypdf",
			"--skip-text",
			srcPath,
			tempOcr,
		)

		if err := cmd.Run(); err == nil {
			workingPath = tempOcr
		}
	}

	if req.Compress {
		compPath := filepath.Join(config.InputDir, "comp_"+finalName)

		cmd := exec.Command(
			"gs",
			"-sDEVICE=pdfwrite",
			"-dCompatibilityLevel=1.4",
			"-dPDFSETTINGS=/ebook",
			"-dNOPAUSE",
			"-dQUIET",
			"-dBATCH",
			"-sOutputFile="+compPath,
			workingPath,
		)

		if err := cmd.Run(); err == nil {
			workingPath = compPath
		}
	}

	err := os.Rename(workingPath, destPath)

	if err != nil {
		if errCopy := utils.CopyFile(workingPath, destPath); errCopy == nil {
			_ = os.Remove(workingPath)
		}
	}

	if workingPath != srcPath && utils.Exists(srcPath) {
		_ = os.Remove(srcPath)
	}

	LastMovedFile = destPath
	LastMovedOrig = filepath.Join(config.InputDir, req.Filename)

	w.WriteHeader(http.StatusOK)
}

func TrashPDF(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")

	if filename == "" {
		return
	}

	src := filepath.Join(config.InputDir, filename)
	dest := filepath.Join(config.TrashDir, filename)

	_ = os.Rename(src, dest)

	LastMovedFile = dest
	LastMovedOrig = src

	w.WriteHeader(http.StatusOK)
}

func UndoLastAction(w http.ResponseWriter, r *http.Request) {
	if LastMovedFile == "" {
		return
	}

	_ = os.Rename(LastMovedFile, LastMovedOrig)

	LastMovedFile = ""

	w.WriteHeader(http.StatusOK)
}

func HandleIndexPage(w http.ResponseWriter, r *http.Request) {
	// Use embedded template instead of reading from file at runtime
	tmpl, err := template.New("index").Parse(templates.IndexHTML)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = tmpl.Execute(w, nil)
}
