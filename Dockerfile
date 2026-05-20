# === STAGE 1: Go-Code autark kompilieren ===
FROM golang:1.21-alpine3.19 AS builder
WORKDIR /build

COPY main.go go.mod ./
COPY vendor/ ./vendor/

# Schaltet CGO aus, optimiert für Linux und baut mit den lokalen Abhängigkeiten
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o pdf-sort main.go


# === STAGE 2: Laufzeit-Image mit deinen fixierten stabilen Versionen ===
FROM alpine:3.19
WORKDIR /app

RUN apk add --no-cache \
    ocrmypdf=15.4.2-r0 \
    poppler-utils=23.10.0-r0 \
    ghostscript=10.05.1-r0 \
    tesseract-ocr-data-deu=5.3.3-r1 \
    tzdata=2025b-r0

COPY --from=builder /build/pdf-sort /app/pdf-sort

EXPOSE 4000
WORKDIR /documents

CMD ["/app/pdf-sort"]
