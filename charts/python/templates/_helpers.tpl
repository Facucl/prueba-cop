{{/*
Set's the container resources if the user has set any.
*/}}
{{- define "api.resources" -}}
  {{- if .Values.api.resources -}}
          resources:
{{ toYaml .Values.api.resources | indent 12}}
  {{- else -}}
          resources:
            requests:
              memory: 256Mi
              cpu: 250m
            limits:
              memory: 256Mi
              cpu: 250m
  {{- end -}}
{{- end -}}

{{/*
Sets extra ingressCore annotations
*/}}
{{- define "api.ingress.annotations" -}}
  {{- if .Values.api.ingress.annotations }}
  annotations:
    {{- tpl .Values.api.ingress.annotations . | nindent 4 }}
  {{- end }}
{{- end -}}

{{- define "core.job" -}}
{{- if .job.enabled -}}
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ .job.name | quote }}
  {{- if .job.annotations }}
  annotations:
{{ toYaml .job.annotations | indent 4 }}
  {{- end }}
spec:
  backoffLimit: {{ .job.backoffLimit }}
  activeDeadlineSeconds: {{ .job.activeDeadlineSeconds }}
  template:
    spec:
      containers:
      - name: {{ .job.containerName | quote }}
        image: {{ .Values.jobs.image.url }}:{{ .Values.jobs.image.tag }}
        command: [ {{ .job.command | quote }}, {{ .job.options | quote }}]
        {{- if .job.resources }}
        resources:
          requests:
            memory: {{ .job.resources.requests.memory | quote }}
            cpu: {{ .job.resources.requests.cpu | quote }}
          limits:
            memory: {{ .job.resources.limits.memory | quote }}
            cpu: {{ .job.resources.limits.cpu | quote }}
        {{ else }}
        resources:
          requests:
            memory: {{ .Values.jobs.resources.requests.memory | quote }}
            cpu: {{ .Values.jobs.resources.requests.cpu | quote }}
          limits:
            memory: {{ .Values.jobs.resources.limits.memory | quote }}
            cpu: {{ .Values.jobs.resources.limits.cpu | quote }}
        {{ end }}
        envFrom:
          - secretRef:
              name: {{ .Values.global.projectName }}-app
      restartPolicy: {{ .job.restartPolicy | quote }}
---
{{- end -}}
{{- end -}}
### ADD CRONJOBS
{{- define "core.cronjob" -}}
{{- if .cronjob.enabled -}}
apiVersion: batch/v1beta1
kind: CronJob
metadata:
  name: {{ .cronjob.name | quote }}
  {{- if .cronjob.annotations }}
  annotations:
{{ toYaml .cronjob.annotations | indent 4 }}
  {{- end }}
spec:
  schedule:  {{ .cronjob.schedule | quote }}
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: {{ .cronjob.containerName | quote }}
            image: {{ .Values.cronjobs.image.url }}:{{ .Values.cronjobs.image.tag }}
            command: ["bash", "-c"]
            args:
            - {{ .cronjob.args | quote }}
            {{- if .cronjob.resources }}
            resources:
              requests:
                memory: {{ .cronjob.resources.requests.memory | quote }}
                cpu: {{ .cronjob.resources.requests.cpu | quote }}
              limits:
                memory: {{ .cronjob.resources.limits.memory | quote }}
                cpu: {{ .cronjob.resources.limits.cpu | quote }}
            {{ else }}
            resources:
              requests:
                memory: {{ .Values.cronjobs.resources.requests.memory | quote }}
                cpu: {{ .Values.cronjobs.resources.requests.cpu | quote }}
              limits:
                memory: {{ .Values.cronjobs.resources.limits.memory | quote }}
                cpu: {{ .Values.cronjobs.resources.limits.cpu | quote }}
            {{ end }}
            envFrom:
              - secretRef:
                  name: {{ .Values.global.projectName }}-app
          restartPolicy: {{ .cronjob.restartPolicy | quote }}
---
{{- end -}}
{{- end -}}
{{/*
Set's the affinity for pod placement.
*/}}
{{- define "api.affinity" -}}
  {{- if .Values.api.affinity }}
      affinity:
        {{ tpl .Values.api.affinity . | nindent 8 | trim }}
  {{ end }}
{{- end -}}

{{/*
Set's the toleration for pod placement when running in standalone and HA modes.
*/}}
{{- define "api.tolerations" -}}
  {{- if .Values.api.tolerations }}
      tolerations:
        {{ tpl .Values.api.tolerations . | nindent 8 | trim }}
  {{- end }}
{{- end -}}

{{/*
Set's the python selector for pod placement when running in standalone and HA modes.
*/}}
{{- define "api.pythonselector" -}}
  {{- if .Values.api.pythonSelector }}
      pythonSelector:
        {{ tpl .Values.api.pythonSelector . | indent 8 | trim }}
  {{- end }}
{{- end -}}



{{/*
Set's up configmap mounts if this isn't a dev deployment and the user
defined a custom configuration.  Additionally iterates over any
extra volumes the user may have specified (such as a secret with TLS).
*/}}
{{- define "api.volumes" -}}
  {{ if .Values.api.extraVolumes }}
      volumes:
    {{- range .Values.api.extraVolumes }}
      - name: {{ .name }}
        {{ .type }}:
        {{- if (eq .type "configMap") }}
          name: {{ .name }}
          defaultMode: {{.defaultMode }}
        {{- else if (eq .type "secret") }}
          secretName: {{ .name }}
        {{- else if (eq .type "persistentVolumeClaim") }}
          claimName: {{ .name }}
        {{- end }}
    {{- end }}
  {{ end }}
{{- end -}}




{{/*
Set's which additional volumes should be mounted to the container
based on the mode configured.
*/}}
{{- define "api.mounts" -}}
  {{ if .Values.api.volumeMounts }}
          volumeMounts:
    {{- range .Values.api.volumeMounts }}
          - name: {{ .name }}
            {{ if .subPath }}
            subPath: {{ .subPath | default "" }}
            {{end}}
            mountPath: {{ .mountPath | default "/mnt" }}
    {{- end }}
  {{- end }}
{{- end -}}


{{/*
Set's which additional enviroment variables should be added to the container.
*/}}
{{- define "api.env" -}}
  {{ if .Values.api.env }}
          env:
    {{- range .Values.api.env }}
            - name: {{ .name }}
              value: {{ .value | quote }}
    {{- end }}
  {{- end }}
{{- end -}}

{{/*
Set's which additional secrets should be added to the container
as enviroment variables.
*/}}
{{- define "api.secretName" -}}
  {{- if .Values.api.envVars.secretVars }}
          {{- range .Values.api.envVars.secretVars }}
          - secretRef:
              name: {{ .name }}
          {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Set's which additional configmaps should be added to the container
as enviroment variables.
*/}}
{{- define "api.configMapName" -}}
  {{- if .Values.api.envVars.configMapVars }}
          {{- range .Values.api.envVars.configMapVars }}
          - configMapRef:
              name: {{ .name }}
          {{- end -}}
  {{- end -}}
{{- end -}}