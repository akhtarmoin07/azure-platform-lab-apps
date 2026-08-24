{{- define "pharmacy-platform.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "pharmacy-platform.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name (include "pharmacy-platform.name" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{- define "pharmacy-platform.labels" -}}
app.kubernetes.io/name: {{ include "pharmacy-platform.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{- end }}

{{- define "pharmacy-platform.selectorLabels" -}}
app.kubernetes.io/name: {{ include "pharmacy-platform.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "pharmacy-platform.migrationJobName" -}}
{{- $imageTag := required "backend.image.tag is required when migration is enabled" .Values.backend.image.tag -}}
{{- $safeImageTag := regexReplaceAll "[^a-z0-9-]+" (lower $imageTag) "-" | trimAll "-" | trunc 12 | trimSuffix "-" -}}
{{- $retryRevision := regexReplaceAll "[^a-z0-9-]+" (lower (toString .Values.migration.retryRevision)) "-" | trimAll "-" -}}
{{- printf "pharmacy-schema-migration-%s-r%s" $safeImageTag $retryRevision | trunc 63 | trimSuffix "-" -}}
{{- end }}
