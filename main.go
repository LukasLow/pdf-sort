package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"

	"pdfsort/src/config"
	"pdfsort/src/handlers"
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

	http.Handle("/static/",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir("src/static")),
		),
	)

	http.HandleFunc("/", handlers.HandleIndexPage)

	fmt.Printf("\n🚀 PDF-Sort gestartet auf %s\n", config.Port)
	log.Fatal(http.ListenAndServe(config.Port, nil))
}
