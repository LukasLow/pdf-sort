# === STAGE 1: Go Build ===
FROM golang:1.21-alpine3.19 AS builder

WORKDIR /build

# Gesamten Source kopieren
COPY . .

# Falls vendor genutzt wird:
RUN go mod vendor

# Build
    RUN CGO_ENABLED=0 GOOS=linux \
        go build -mod=vendor -o pdf-sort . && \
        # Run tests in the builder where Go is available
        go test ./src/services/...

# =========================================================

# === STAGE 2: Runtime ===
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache \
    ocrmypdf=15.4.2-r0 \
    poppler-utils=23.10.0-r0 \
    ghostscript=10.05.1-r0 \
    tesseract-ocr-data-deu=5.3.3-r1 \
    tzdata=2025b-r0

# Binary kopieren
COPY --from=builder /build/pdf-sort /app/pdf-sort

# Templates + Static Files kopieren
COPY --from=builder /build/src/templates /app/src/templates
COPY --from=builder /build/src/static /app/src/static
    

EXPOSE 4000

WORKDIR /documents

CMD ["/app/pdf-sort"]
