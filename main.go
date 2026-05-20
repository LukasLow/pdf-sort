package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"pdf-sort/src/config"
	"pdf-sort/src/embed"
	"pdf-sort/src/handlers"
)

func main() {
	_, err := exec.LookPath("ocrmypdf")
	if err != nil {
		log.Fatalf("❌ FEHLER: 'ocrmypdf' wurde nicht gefunden!")
	}

	for _, dir := range []string{
		config.InputDir,
		config.TrashDir,
		config.BaseArch,
	} {
		_ = os.MkdirAll(dir, 0755)
	}

	if _, err := os.Stat(config.YamlPath); os.IsNotExist(err) {
		_ = os.WriteFile(config.YamlPath, []byte("{}"), 0644)
	}

	http.HandleFunc("/next-pdf", handlers.GetNextPDF)
	http.HandleFunc("/add-entry", handlers.AddNewEntry)
	http.HandleFunc("/process", handlers.ProcessPDF)
	http.HandleFunc("/trash", handlers.TrashPDF)
	http.HandleFunc("/undo", handlers.UndoLastAction)

	http.Handle(
		"/view-pdf/",
		http.StripPrefix(
			"/view-pdf/",
			http.FileServer(http.Dir(config.InputDir)),
		),
	)

	// Serve static assets from the embedded filesystem
	staticFS, _ := fs.Sub(embed.StaticFiles, "static")
	http.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))),
	)

	// Serve the embedded index.html at the root path
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        data, err := embed.StaticFiles.ReadFile("static/index.html")
        if err != nil {
            http.Error(w, "index not found", http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        _, _ = w.Write(data)
    })

	fmt.Printf("\n🚀 PDF-Sort gestartet auf %s\n", config.Port)
	log.Fatal(http.ListenAndServe(config.Port, nil))
}

// BuildTime is injected at compile time via -ldflags "-X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
var BuildTime string = ""

func init() {
	if BuildTime == "" {
		// Fallback to current time if not set during build
		BuildTime = time.Now().UTC().Format(time.RFC3339)
	}
	http.HandleFunc("/buildinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "{\"build\": \"%s\"}", BuildTime)
	})
}
