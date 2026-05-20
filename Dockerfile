# === STAGE 1: Go Build ===
# Wir nutzen TARGETPLATFORM Variablen, daher ist die ARGS-Definition wichtig
FROM --platform=$BUILDPLATFORM golang:1.21-alpine3.19 AS builder

# Docker setzt diese Variablen bei Multi-Arch-Builds automatisch
ARG TARGETOS
ARG TARGETARCH

WORKDIR /build

# Gesamten Source kopieren
COPY . .

# Falls vendor genutzt wird:
RUN go mod vendor

# Build: Hier nutzen wir jetzt TARGETOS und TARGETARCH dynamisch
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -mod=vendor -o pdf-sort .

# (Die Tests wurden hier entfernt, da sie beim Cross-Compiling für eine 
# andere Architektur fehlschlagen würden. Tests am besten in der Pipeline vor dem Build ausführen!)

# =========================================================

# === STAGE 2: Runtime ===
FROM alpine:3.19

WORKDIR /app

# Paketversionen ohne die feste Revision (-r0), damit es auf AMD64 und ARM64 sauber durchläuft
RUN apk add --no-cache \
    ocrmypdf \
    poppler-utils \
    ghostscript \
    tesseract-ocr-data-deu \
    tzdata

# Binary kopieren
COPY --from=builder /build/pdf-sort /app/pdf-sort

# Templates + Static Files kopieren
COPY --from=builder /build/src/templates /app/src/templates
COPY --from=builder /build/src/static /app/src/static

EXPOSE 4000

WORKDIR /documents

CMD ["/app/pdf-sort"]