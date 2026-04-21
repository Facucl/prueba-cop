# RESUME — Bot de notificaciones de despliegues (Copérnico)

Este documento captura el estado completo del proyecto al 2026-04-21 para
retomar la conversación en el mismo punto donde quedó.

---

## 1. Qué es este bot

Notifica al canal de Microsoft Teams `COPÉRNICO › DEPLOYS (BOT)` el resultado
de despliegues disparados por **fixes de seguridad automáticos** sobre
servicios que usan el chart `node-pipeline`.

- Requerimiento: `BOT_NOTIFICACIONES_DESPLIEGUES_CON_FIXES_DE_SEGURIDAD (1).pdf`
  (Martín Castagnolo, 2026-04-20).
- Canal Teams: `COPÉRNICO › DEPLOYS (BOT)` (webhook vía relay
  `webhookbot.c-toss.com`, **URL debe ser rotada** — fue expuesta en chat).
- Servicio de ejemplo: `ws-copernico-suite-back-cop`.

## 2. Arquitectura del flujo

```
tag annotated en GitLab (con "deploy: desa test preprod" en body)
  → EventListener (Tekton)
  → PipelineRun (SA: tekton-build, Pipeline: <release>-pipeline)
  → tasks: git-clone → build-push-kaniko → git-cli (sed values, push) → trivy
  → [FIN DEL PIPELINE PRINCIPAL]
  → Argo sincroniza async en cada cluster remoto (DESA/TEST/PREPROD/PROD)
  → finally: pod bot (Go) polla Applications → webhook Teams
```

**Stack confirmado:**
- Tekton Pipelines en OpenShift (no GitLab CI)
- ArgoCD en el mismo cluster CI/CD, ns `openshift-gitops`, sync a clusters
  remotos DESA/TEST/PREPROD/PROD
- Chart `node-pipeline` es **compartido** por ~60 servicios (ant-cop, gab-cop,
  backftago, ws-copernico-suite-back-cop, etc.) en
  `git.rentasweb.gob.ar/DevOps/helm-charts.git`
- Chart de servicio es wrapper del chart base `node` (via `dependencies`)
- Probes habilitadas SOLO en PROD → `Healthy` en DESA/TEST/PREPROD sólo
  significa "Deployment terminó el rollout", no "app responde OK"

## 3. Decisiones tomadas (todas con OK explícito del usuario)

| # | Decisión | Razón |
|---|---|---|
| 1 | `finally` **universal** en `charts/node-pipeline/templates/2-pipeline.yaml` | La guardia anti-ruido es el subject del commit. Si no matchea, `exit 0`. |
| 2 | Formato del mensaje: `{"text": "\`\`\`...\`\`\`"}` con markdown + code block | Relay `c-toss` sólo acepta `text`; el PDF muestra texto monoespaciado |
| 3 | DESCRIPCIÓN = primera línea del commit message (prefijo `· ` U+00B7) | Decisión de diseño — el subject del commit |
| 4 | RAMA = hardcoded `master` por ahora | Mayoría de casos serán master. Deducir con `git branch -r --contains` queda pendiente |
| 5 | Poll: `15s` interval, `15m` timeout | Equilibrio cache-warm / tiempo típico de rollout |
| 6 | Sin whitelist de envs — notifica lo que liste `deploy:` (incluye prod) | Flexibilidad |
| 7 | SERVICIO con sufijo `-cop` tal cual el `ecr-repository` | Evita transformaciones de nombre |
| 8 | `finally` bloqueante (NO fire-and-forget) | El deploy real ocurre en step 3 (`git-cli`), no se retrasa. Una sola PipelineRun = una unidad |

## 4. Archivos del proyecto

```
prueba-cop/   ← este repo
├── bot/
│   ├── Dockerfile                          distroless, nonroot
│   ├── go.mod                              k8s.io/client-go v0.30.3
│   ├── cmd/bot/main.go                     orquesta: guard → pipeline status → poll → teams
│   ├── internal/
│   │   ├── config/config.go                env vars + defaults (15s/15m, branch=master)
│   │   ├── tag/parser.go                   ParseList, ParseRequestedEnvs, SortCanonical
│   │   ├── tag/parser_test.go              17 casos
│   │   ├── argo/client.go                  dynamic client, polla applications.argoproj.io
│   │   ├── message/builder.go              5 escenarios, payload {"text":"```...```"}
│   │   ├── message/builder_test.go         7 casos verificando cada field
│   │   └── teams/webhook.go                POST simple
│   └── deploy/
│       ├── notify-task.yaml                Task standalone (para install manual)
│       ├── pipeline-finally.yaml           doc — el finally efectivo vive en el chart
│       └── rbac.yaml                       Role + RoleBinding (SA tekton-build)
├── chart-changes/                          cambios para aplicar al repo helm-charts externo
│   ├── README.md                           instrucciones de aplicación
│   └── node-pipeline/templates/
│       ├── 6-notify-task.yaml              archivo nuevo
│       └── 2-pipeline.yaml                 existente con bloque finally agregado
└── RESUME.md                               este archivo
```

## 5. Estado de validación

- `go vet ./...` — ✅ limpio
- `go build ./...` — ✅ limpio
- `go test ./...` — ✅ 19/19 tests pasan
- Render manual de los 5 escenarios del PDF — ✅ matchean

## 6. Lo que falta (fuera del alcance de esta iteración)

1. **Build + push de la imagen del bot** a `registry-sip.cba.gov.ar:5000/deploy-notify-bot:<tag>`.
   Requiere pipeline propio del bot o build manual.
2. **Crear el Secret `deploy-notify-bot-teams`** en cada ns `-pipeline` de los
   servicios que usen `node-pipeline`, con la URL de Teams **rotada**
   (la original quedó expuesta y debe reemplazarse).
3. **Aplicar los cambios de `chart-changes/`** al repo real
   `git.rentasweb.gob.ar/DevOps/helm-charts.git`.
4. **Deducción de rama** (task futura): leer con `git branch -r --contains $SHA`
   en el step `read-commit-message`. Orden de preferencia: `master > main >
   primera alfabética`. Si no hay match, usar `"(desconocida)"`.
5. **Probar end-to-end** con un tag real de prueba en un servicio que use
   `node-pipeline`, con commit que matchee la marca de seguridad.

## 7. Contactos y convenciones

- Autor del requerimiento: Martín Castagnolo, Senior Backend Developer
- Registry interno: `registry-sip.cba.gov.ar:5000`
- ArgoCD Applications:
  - ns: `openshift-gitops`
  - naming: `<ecr-repository>-<env>` → `ws-copernico-suite-back-cop-desa`, etc.
- AppProject: `rcb-gitops-projects-<app-name>`
- Convención values: `values-<env>.yaml` **excepto prod** que es `values-prod-os.yaml`
- Tag trigger: commit subject debe ser exactamente
  `Mantenimiento: Fixes de seguridad y reparación automática de dependencias.`
  (con tildes y punto final).
- Leer el subject: `git log -1 --format=%B $CI_COMMIT_SHA` (en un pipeline de
  tag, `CI_COMMIT_MESSAGE` trae el mensaje del tag, no del commit).
