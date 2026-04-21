# chart-changes/

Cambios al chart compartido `node-pipeline` que viven en el repo externo
`git.rentasweb.gob.ar/DevOps/helm-charts.git`. Acá quedan como copia de
referencia — NO son el lugar donde se aplican en producción.

## Qué hay

- `node-pipeline/templates/6-notify-task.yaml` — **archivo nuevo**.
  Copiar tal cual a `charts/node-pipeline/templates/6-notify-task.yaml` en el
  repo `helm-charts`.
- `node-pipeline/templates/2-pipeline.yaml` — **archivo existente con un
  bloque `finally:` agregado al final del `spec:`**. Hay que mergear el bloque
  nuevo; buscar la sección `# FINALLY — notificar a Teams el resultado del
  despliegue` que se agrega después del último task (`trivy-scan-repo`).

## Cómo aplicarlo al repo real

```
# En un clone del helm-charts:
cp /ruta/a/este/repo/chart-changes/node-pipeline/templates/6-notify-task.yaml \
   charts/node-pipeline/templates/
# Para 2-pipeline.yaml, abrir los dos en un diff visual y copiar el bloque
# `finally:` — NO reemplazar el archivo entero porque puede haber cambios
# propios del equipo de DevOps que no están acá.
git add charts/node-pipeline/templates/
git commit -m "feat(node-pipeline): finally notify-deploy para bot de Teams"
```

## Impacto

Al mergear el chart, **todos los servicios que hoy usan `node-pipeline`**
(~60 servicios) pasan a tener el `finally` activo. La guardia que evita
notificar en deploys normales es el mensaje del commit: el bot sale con
`exit 0` si el commit no contiene exactamente la frase
`Mantenimiento: Fixes de seguridad y reparación automática de dependencias.`.

Por eso es seguro hacerlo universal — sólo los deploys automáticos del bot
de fixes de seguridad disparan la notificación.
