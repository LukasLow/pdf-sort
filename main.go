package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config map[string]struct {
	Path string   `yaml:"Path"`
	Info []string `yaml:"INFO"`
}

type ProcessRequest struct {
	Filename      string `json:"filename"`
	Year          string `json:"year"`
	Month         string `json:"month"`
	Day           string `json:"day"`
	Correspondent string `json:"correspondent"`
	Info          string `json:"info"`
	Extra         string `json:"extra"`
	Compress      bool   `json:"compress"`
}

type NewEntryRequest struct {
	Correspondent string `json:"correspondent"`
	Info          string `json:"info"`
}

type NextResponse struct {
	Filename      string              `json:"filename"`
	Year          string              `json:"year"`
	Month         string              `json:"month"`
	Day           string              `json:"day"`
	SuggestedCorr string              `json:"suggested_correspondent"`
	SuggestedInfo string              `json:"suggested_info"`
	ConfigData    map[string][]string `json:"config_data"`
}

// Pfade angepasst auf das starr gemountete /documents Verzeichnis im Container
const (
	InputDir = "/documents/900-eingang"
	TrashDir = "/documents/900-Trash"
	BaseArch = "/documents/Archiv"
	YamlPath = "/documents/korespondenten.yml"
	Port     = ":4000"
)

var lastMovedFile string
var lastMovedOrig string

func main() {
	// Überprüfung auf ocrmypdf im Container
	_, err := exec.LookPath("ocrmypdf")
	if err != nil {
		log.Fatalf("❌ FEHLER: 'ocrmypdf' wurde im Container nicht gefunden!")
	}

	// Ordnerstruktur im Container-Mount anlegen
	for _, dir := range []string{InputDir, TrashDir, BaseArch} {
		_ = os.MkdirAll(dir, 0755)
	}
	if _, err := os.Stat(YamlPath); os.IsNotExist(err) {
		_ = os.WriteFile(YamlPath, []byte("{}"), 0644)
	}

	http.HandleFunc("/next-pdf", getNextPDF)
	http.HandleFunc("/add-entry", addNewEntry)
	http.HandleFunc("/process", processPDF)
	http.HandleFunc("/trash", trashPDF)
	http.HandleFunc("/undo", undoLastAction)

	// PDFs für das Web-Interface bereitstellen
	http.Handle("/view-pdf/", http.StripPrefix("/view-pdf/", http.FileServer(http.Dir(InputDir))))

	// Integriertes HTML-Template rendern
	http.HandleFunc("/", handleIndexPage)

	fmt.Printf("\n🚀 PDF-Sort-Backend im Container gestartet auf Port %s\n", Port)
	log.Fatal(http.ListenAndServe(Port, nil))
}

func loadConfig() Config {
	var cfg Config
	data, err := os.ReadFile(YamlPath)
	if err != nil {
		return make(Config)
	}
	_ = yaml.Unmarshal(data, &cfg)
	if cfg == nil {
		return make(Config)
	}
	return cfg
}

func saveConfig(cfg Config) {
	data, _ := yaml.Marshal(&cfg)
	_ = os.WriteFile(YamlPath, data, 0644)
}

func extractText(path string) string {
	cmd := exec.Command("pdftotext", "-l", "1", path, "-")
	var out bytes.Buffer
	cmd.Stdout = &out
	_ = cmd.Run()
	return out.String()
}

func findDate(text string) (y, m, d string) {
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

	months := map[string]string{
		"januar": "01", "jan": "01", "january": "01",
		"februar": "02", "feb": "02", "february": "02",
		"märz": "03", "mar": "03", "march": "03", "mrz": "03",
		"april": "04", "apr": "04", "mai": "05", "may": "05",
		"juni": "06", "jun": "06", "june": "06", "juli": "07",
		"jul": "07", "july": "07", "august": "08", "aug": "08",
		"september": "09", "sep": "09", "oktober": "10", "okt": "10",
		"october": "10", "oct": "10", "november": "11", "nov": "11",
		"dezember": "12", "dez": "12", "december": "12", "dec": "12",
	}

	reTextual := regexp.MustCompile(`\b(\d{1,2})\.\s*([a-zA-ZäÄöÖüÜß]+)\s*(\d{4})\b`)
	if match := reTextual.FindStringSubmatch(text); match != nil {
		monLower := strings.ToLower(match[2])
		if num, ok := months[monLower]; ok {
			return match[3], num, fmt.Sprintf("%02s", match[1])
		}
	}
	return "", "", ""
}

func getNextPDF(w http.ResponseWriter, r *http.Request) {
	files, _ := filepath.Glob(filepath.Join(InputDir, "*.pdf"))
	if len(files) == 0 {
		json.NewEncoder(w).Encode(NextResponse{})
		return
	}
	sort.Strings(files)
	targetFile := files[0]
	filename := filepath.Base(targetFile)

	text := extractText(targetFile)
	y, m, d := findDate(text)

	cfg := loadConfig()
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
	json.NewEncoder(w).Encode(NextResponse{
		Filename:      filename,
		Year:          y,
		Month:         m,
		Day:           d,
		SuggestedCorr: suggestedCorr,
		SuggestedInfo: suggestedInfo,
		ConfigData:    uiConfig,
	})
}

func addNewEntry(w http.ResponseWriter, r *http.Request) {
	var req NewEntryRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.Correspondent == "" {
		http.Error(w, "Name fehlt", http.StatusBadRequest)
		return
	}

	cfg := loadConfig()
	val, ok := cfg[req.Correspondent]
	if !ok {
		val.Path = filepath.Join(BaseArch, req.Correspondent)
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
	saveConfig(cfg)
	w.WriteHeader(http.StatusOK)
}

func processPDF(w http.ResponseWriter, r *http.Request) {
	var req ProcessRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	srcPath := filepath.Join(InputDir, req.Filename)
	cfg := loadConfig()

	val, ok := cfg[req.Correspondent]
	if !ok {
		http.Error(w, "Korrespondent existiert nicht", http.StatusBadRequest)
		return
	}

	finalDestDir := filepath.Join(val.Path, req.Info)
	_ = os.MkdirAll(finalDestDir, 0755)

	finalName := fmt.Sprintf("%s-%s-%s_%s_%s", req.Year, req.Month, req.Day, req.Correspondent, req.Info)
	if req.Extra != "" {
		finalName += "_" + req.Extra
	}
	finalName += ".pdf"
	destPath := filepath.Join(finalDestDir, finalName)

	tempOcr := filepath.Join(InputDir, "ocr_"+finalName)
	text := extractText(srcPath)
	workingPath := srcPath

	if len(strings.TrimSpace(text)) == 0 {
		cmd := exec.Command("ocrmypdf", "--skip-text", srcPath, tempOcr)
		if err := cmd.Run(); err == nil {
			workingPath = tempOcr
		}
	}

	if req.Compress {
		compPath := filepath.Join(InputDir, "comp_"+finalName)
		cmd := exec.Command("gs", "-sDEVICE=pdfwrite", "-dCompatibilityLevel=1.4", "-dPDFSETTINGS=/ebook", "-dNOPAUSE", "-dQUIET", "-dBATCH", "-sOutputFile="+compPath, workingPath)
		if err := cmd.Run(); err == nil {
			if workingPath == tempOcr {
				_ = os.Remove(tempOcr)
			}
			workingPath = compPath
		}
	}

	err := os.Rename(workingPath, destPath)
	if err != nil {
		if errCopy := copyFile(workingPath, destPath); errCopy == nil {
			_ = os.Remove(workingPath)
		}
	}
	// HIER WURDE DER FEHLER BEHOBEN: von os.Exists zu osExists gewechselt
	if workingPath != srcPath && osExists(srcPath) {
		_ = os.Remove(srcPath)
	}

	lastMovedFile = destPath
	lastMovedOrig = filepath.Join(InputDir, req.Filename)

	w.WriteHeader(http.StatusOK)
}

func trashPDF(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")
	if filename == "" { return }
	src := filepath.Join(InputDir, filename)
	dest := filepath.Join(TrashDir, filename)

	_ = os.Rename(src, dest)

	lastMovedFile = dest
	lastMovedOrig = src
	w.WriteHeader(http.StatusOK)
}

func undoLastAction(w http.ResponseWriter, r *http.Request) {
	if lastMovedFile == "" { return }
	_ = os.Rename(lastMovedFile, lastMovedOrig)
	lastMovedFile = ""
	w.WriteHeader(http.StatusOK)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil { return err }
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil { return err }
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// HIER: Funktion leicht umbenannt, um Namenskonflikte mit dem 'os'-Paket zu vermeiden
func osExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func handleIndexPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = tmpl.Execute(w, nil)
}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="de">
<head>
    <meta charset="UTF-8">
    <title>PDF Archivierungs-Zentrale</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; margin: 0; padding: 0; background: #f5f5f7; display: flex; height: 100vh; overflow: hidden; }
        .preview-side { flex: 1; background: #525659; display: flex; align-items: center; justify-content: center; }
        .preview-side iframe { width: 100%; height: 100%; border: none; }
        .form-side { width: 460px; padding: 25px; background: white; box-shadow: -4px 0 12px rgba(0,0,0,0.05); display: flex; flex-direction: column; justify-content: space-between; overflow-y: auto; box-sizing: border-box;}
        h1 { margin-top: 0; font-size: 20px; color: #1d1d1f; }
        h2 { font-size: 14px; color: #1d1d1f; margin-top: 0; margin-bottom: 12px;}
        .field { margin-bottom: 14px; position: relative; }
        label { display: block; font-weight: 600; font-size: 13px; margin-bottom: 6px; color: #515154; }

        .section-box { background: #f5f5f7; border: 1px solid #d2d2d7; padding: 14px; border-radius: 8px; margin-bottom: 20px; }
        .inline-group { display: flex; gap: 8px; margin-bottom: 8px; }

        .date-group { display: flex; gap: 8px; }
        .date-group input { text-align: center; font-size: 16px; padding: 10px; border: 1px solid #d2d2d7; border-radius: 6px; }
        .w-year { flex: 2; } .w-mon, .w-day { flex: 1; }

        input[type="text"], select { width: 100%; padding: 11px; border: 1px solid #d2d2d7; border-radius: 6px; font-size: 16px; box-sizing: border-box; background: white;}
        .live-preview { background: #e8e8ed; padding: 12px; border-radius: 6px; font-family: monospace; font-size: 13px; color: #1d1d1f; word-break: break-all; margin-top: 5px; min-height: 16px;}

        .btn-row { display: flex; gap: 10px; margin-top: 10px;}
        button { color: white; border: none; padding: 12px; border-radius: 8px; font-size: 14px; cursor: pointer; font-weight: 600; flex: 1;}
        .btn-main { background: #0071e3; flex: 2;} .btn-main:hover { background: #0077ed; }
        .btn-trash { background: #ff453a;} .btn-trash:hover { background: #ff5247;}
        .btn-undo { background: #86868b;} .btn-undo:hover { background: #98989d;}
        .btn-add { background: #34c759; padding: 10px; font-size: 13px;} .btn-add:hover { background: #30b351; }
        button:disabled { background: #aeaeaf; cursor: not-allowed; }

        #status { margin-top: 10px; font-size: 14px; color: #0071e3; text-align: center; font-weight: 500; min-height: 20px;}
    </style>
</head>
<body>

    <div class="preview-side">
        <iframe id="pdf-viewer" src=""></iframe>
    </div>

    <div class="form-side">
        <div>
            <h1>📄 Archivierungs-Zentrale</h1>
            <hr style="border: 0; border-top: 1px solid #e5e5ea; margin-bottom: 15px;">

            <div class="section-box">
                <h2>➕ Neuen Eintrag zur Liste hinzufügen</h2>
                <div class="inline-group">
                    <input type="text" id="new-corr" placeholder="Neuer Korrespondent...">
                    <button class="btn-add" onclick="submitNewConfigEntry(true)">Anlegen</button>
                </div>
                <div class="inline-group">
                    <select id="new-info-parent"></select>
                    <input type="text" id="new-info-text" placeholder="Neue Info für gewählten Korr...">
                    <button class="btn-add" onclick="submitNewConfigEntry(false)">Anlegen</button>
                </div>
            </div>

            <div class="field">
                <label>Dokumenten-Datum (Jahr / Monat / Tag)</label>
                <div class="date-group">
                    <input type="text" id="date-year" class="w-year" placeholder="YYYY" maxlength="4" oninput="updatePreview()">
                    <input type="text" id="date-month" class="w-mon" placeholder="MM" maxlength="2" oninput="updatePreview()">
                    <input type="text" id="date-day" class="w-day" placeholder="DD" maxlength="2" oninput="updatePreview()">
                </div>
            </div>

            <div class="field">
                <label>Korrespondent</label>
                <select id="sel-correspondent" onchange="onCorrespondentChange()"></select>
            </div>

            <div class="field">
                <label>Info</label>
                <select id="sel-info" onchange="updatePreview()"></select>
            </div>

            <div class="field">
                <label>EXTRA (Freitext für Einmaliges)</label>
                <input type="text" id="extra-text" placeholder="Zusatz für Dateiende..." oninput="updatePreview()">
            </div>

            <div class="field" style="margin-top: 15px;">
                <label style="display: flex; align-items: center; gap: 10px; cursor: pointer; font-size: 13px;">
                    <input type="checkbox" id="compress"> 🗜️ PDF zusätzlich komprimieren
                </label>
            </div>
        </div>

        <div>
            <div class="field">
                <label>Vorschau Dateiname:</label>
                <div id="filename-preview" class="live-preview">...</div>
            </div>

            <div class="btn-row">
                <button id="submit-btn" class="btn-main" onclick="processFile()" disabled>Archivieren</button>
                <button id="trash-btn" class="btn-trash" onclick="trashFile()" disabled>🗑️ Müll</button>
            </div>
            <div class="btn-row">
                <button id="undo-btn" class="btn-undo" onclick="undoAction()" style="display: none;">↩️ Rückgängig (Letzte Datei)</button>
            </div>
            <div id="status"></div>
        </div>
    </div>

<script>
    let currentFile = "";
    let globalConfig = {};

    function updatePreview() {
        const year = document.getElementById('date-year').value.trim();
        const month = document.getElementById('date-month').value.trim();
        const day = document.getElementById('date-day').value.trim();
        const corr = document.getElementById('sel-correspondent').value;
        const info = document.getElementById('sel-info').value;
        const extra = document.getElementById('extra-text').value.trim();

        let preview = "";
        if(year || month || day) {
            preview += year + "-" + month + "-" + day;
        } else {
            preview += "JJJJ-MM-TT";
        }
        preview += "_" + (corr || "KORRESPONDENT") + "_" + (info || "INFO");
        if(extra) preview += "_" + extra;
        preview += ".pdf";

        document.getElementById('filename-preview').innerText = preview;
    }

    function onCorrespondentChange() {
        const corr = document.getElementById('sel-correspondent').value;
        const infoSelect = document.getElementById('sel-info');
        infoSelect.innerHTML = "";

        if (globalConfig[corr]) {
            globalConfig[corr].forEach(info => {
                let opt = document.createElement('option');
                opt.value = info; opt.innerText = info;
                infoSelect.appendChild(opt);
            });
        }
        updatePreview();
    }

    async function submitNewConfigEntry(isCorr) {
        let payload = { correspondent: "", info: "" };
        if(isCorr) {
            payload.correspondent = document.getElementById('new-corr').value.trim();
            if(!payload.correspondent) return;
        } else {
            payload.correspondent = document.getElementById('new-info-parent').value;
            payload.info = document.getElementById('new-info-text').value.trim();
            if(!payload.correspondent || !payload.info) return;
        }

        const res = await fetch('/add-entry', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify(payload)
        });

        if(res.ok) {
            document.getElementById('new-corr').value = "";
            document.getElementById('new-info-text').value = "";

            const fileRes = await fetch('/next-pdf');
            const data = await fileRes.json();
            globalConfig = data.config_data || {};
            renderDropdowns(document.getElementById('sel-correspondent').value, document.getElementById('sel-info').value);
        }
    }

    function renderDropdowns(selectedCorr, selectedInfo) {
        const corrSel = document.getElementById('sel-correspondent');
        const parentSel = document.getElementById('new-info-parent');

        corrSel.innerHTML = "";
        parentSel.innerHTML = "";

        Object.keys(globalConfig).sort().forEach(corr => {
            let opt1 = document.createElement('option'); opt1.value = corr; opt1.innerText = corr;
            let opt2 = document.createElement('option'); opt2.value = corr; opt2.innerText = corr;
            corrSel.appendChild(opt1);
            parentSel.appendChild(opt2);
        });

        if(selectedCorr && globalConfig[selectedCorr]) {
            corrSel.value = selectedCorr;
        }

        onCorrespondentChange();

        if(selectedInfo) {
            document.getElementById('sel-info').value = selectedInfo;
        }
        updatePreview();
    }

    async function loadNextFile() {
        document.getElementById('status').innerText = "Lade Dokument...";
        try {
            const response = await fetch('/next-pdf');
            const data = await response.json();

            globalConfig = data.config_data || {};

            if (data.filename) {
                currentFile = data.filename;
                document.getElementById('pdf-viewer').src = "/view-pdf/" + encodeURIComponent(currentFile);

                document.getElementById('date-year').value = data.year;
                document.getElementById('date-month').value = data.month;
                document.getElementById('date-day').value = data.day;

                renderDropdowns(data.suggested_correspondent, data.suggested_info);

                document.getElementById('extra-text').value = "";
                document.getElementById('submit-btn').disabled = false;
                document.getElementById('trash-btn').disabled = false;
                document.getElementById('status').innerText = "Bereit.";
            } else {
                document.getElementById('pdf-viewer').src = "about:blank";
                document.getElementById('filename-preview').innerText = "Keine PDFs im Ordner.";
                renderDropdowns("", "");
                document.getElementById('date-year').value = "";
                document.getElementById('date-month').value = "";
                document.getElementById('date-day').value = "";
                document.getElementById('extra-text').value = "";
                document.getElementById('submit-btn').disabled = true;
                document.getElementById('trash-btn').disabled = true;
                document.getElementById('status').innerText = "🎉 Fertig! Der Eingang ist leer.";
            }
        } catch (e) {
            document.getElementById('status').innerText = "Fehler: Go-Server läuft nicht.";
        }
    }

    async function processFile() {
        const year = document.getElementById('date-year').value.trim();
        const month = document.getElementById('date-month').value.trim();
        const day = document.getElementById('date-day').value.trim();
        const corr = document.getElementById('sel-correspondent').value;
        const info = document.getElementById('sel-info').value;
        const extra = document.getElementById('extra-text').value.trim();
        const compress = document.getElementById('compress').checked;

        if(!year || !month || !day || !corr || !info) {
            alert("Bitte Datum, Korrespondent und Info auswählen!");
            return;
        }

        document.getElementById('status').innerText = "Verarbeite (OCR / Speichern)...";
        document.getElementById('submit-btn').disabled = true;

        try {
            const res = await fetch('/process', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({ filename: currentFile, year, month, day, correspondent: corr, info, extra, compress })
            });
            if(res.ok) {
                document.getElementById('compress').checked = false;
                document.getElementById('undo-btn').style.display = "block";
                loadNextFile();
            } else {
                alert("Fehler beim Speichern.");
                document.getElementById('submit-btn').disabled = false;
            }
        } catch (e) {
            alert("Serverfehler.");
        }
    }

    async function trashFile() {
        if(!confirm("Datei nach './900-Trash' verschieben?")) return;
        await fetch("/trash?filename=" + encodeURIComponent(currentFile));
        document.getElementById('undo-btn').style.display = "block";
        loadNextFile();
    }

    async function undoAction() {
        const res = await fetch('/undo');
        if(res.ok) {
            document.getElementById('undo-btn').style.display = "none";
            document.getElementById('status').innerText = "Aktion rückgängig gemacht!";
            loadNextFile();
        }
    }

    loadNextFile();
</script>
</body>
</html>
`
