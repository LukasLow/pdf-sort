# Variablen für den Image-Namen
IMAGE_NAME = ghcr.io/lukaslow/pdf-sort
TAG = latest

.PHONY: all test run clean

# Standardbefehl: Wenn du nur "make" tippst, wird das Image gebaut
all: test

# Das Herzstück: Baut das Dockerfile komplett inklusive Go-Kompilierung
test:
	@echo "⏳ Starte Test-Build... Überprüfe Go-Code und Docker-Struktur..."
	docker build -t $(IMAGE_NAME):$(TAG) .
	@echo "✅ TEST ERFOLGREICH: Go kompiliert fehlerfrei und das Docker-Image steht bereit!"

# Lokaler Testlauf (simuliert den CLI-Befehl ohne Browser-Start)
run:
	@echo "🚀 Starte Container lokal auf Port 4000..."
	docker run --rm -it \
		-p 4000:4000 \
		-v "$$(pwd):/documents" \
		$(IMAGE_NAME):$(TAG)

# Aufräumen von ungenutzten Docker-Resten
clean:
	@echo "🧹 Räume Docker-Images auf..."
	docker rmi $(IMAGE_NAME):$(TAG) || true
	docker image prune -f
