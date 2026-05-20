docker run --pull=always --rm -it -p 4000:4000 -v "$(pwd):/documents" ghcr.io/lukaslow/pdf-sort:latest


docker run --rm -it -p 4000:4000 -v "$(pwd):/documents" ghcr.io/lukaslow/pdf-sort:latest



'''sh
alias pdf-sort='docker run --rm -it -p 4000:4000 -v "$(pwd):/documents" ghcr.io/lukaslow/pdf-sort:latest'
''


'''sh
function pdf-sort() {
    # 1. Browser im Hintergrund nach 1 Sekunde öffnen
    (sleep 1 && open "http://localhost:4000") &
    
    # 2. Den Docker-Container interaktiv starten
    docker run --rm -it \
        -p 4000:4000 \
        -v "$(pwd):/documents" \
        ghcr.io/lukaslow/pdf-sort:latest
}'''


# For Nixos
