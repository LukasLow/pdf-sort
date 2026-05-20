{ pkgs, ... }:

pkgs.writeShellApplication {
  name = "pdf-sort";

  # Falls du sicherstellen willst, dass Docker installiert ist,
  # kannst du es hier in die Laufzeit-Abhängigkeiten aufnehmen.
  runtimeInputs = [ pkgs.docker ];

  text = ''
    # Prüfen, welches Betriebssystem läuft, um den richtigen Browser-Befehl zu wählen
    if command -v open >/dev/null 2>&1; then
      BROWSER_CMD="open"        # macOS Standard
    elif command -v xdg-open >/dev/null 2>&1; then
      BROWSER_CMD="xdg-open"    # Linux / NixOS Standard
    else
      BROWSER_CMD="echo Bitte öffne:"
    fi

    # 1. Browser im Hintergrund nach einer Sekunde öffnen
    (sleep 1 && $BROWSER_CMD "http://localhost:4000") &

    # 2. Den Docker-Container im aktuellen Verzeichnis ausführen
    # $（pwd）wird durch Nix sicher zur Laufzeit ausgewertet
    docker run --rm -it \
        -p 4000:4000 \
        -v "$(pwd):/documents" \
        ghcr.io/lukaslow/pdf-sort:latest
  '';
}
