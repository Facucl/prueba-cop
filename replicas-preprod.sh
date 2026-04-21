#!/usr/bin/env bash
set -euo pipefail

# Directorio base donde están tus charts (por defecto "charts")
BASE_DIR="${1:-charts}"

find "$BASE_DIR" -type f -name 'values-preprod.yaml' | while read -r file; do
  # Nombre de la carpeta del chart (ej: wsrestmiapp)
  chart_dir="$(basename "$(dirname "$file")")"

  # Buscar la primera línea con replicaCount (ignorando líneas comentadas)
  replica="$(awk '
    /^[[:space:]]*#/ {next}  # ignorar comentarios
    /replicaCount[[:space:]]*:/ {
      for (i = 1; i <= NF; i++) {
        if ($i ~ /replicaCount[[:space:]]*:/) {
          # Quitar "replicaCount:" y espacios
          gsub(/replicaCount[[:space:]]*:/, "", $i)
          gsub(/#.*/, "", $i)
          gsub(/[[:space:]]*/, "", $i)
          if ($i == "") {
            print $(i+1)  # caso "replicaCount: 2"
          } else {
            print $i      # caso "replicaCount:2"
          }
          exit
        }
      }
    }
  ' "$file")"

  # Mostrar solo los que tienen más de 1 réplica
  if [[ -n "$replica" && "$replica" =~ ^[0-9]+$ && "$replica" -gt 1 ]]; then
    echo "$chart_dir - cantidad de replicas \"$replica\""
  fi
done
