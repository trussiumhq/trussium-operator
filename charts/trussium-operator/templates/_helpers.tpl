{{- define "trussium-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- define "trussium-operator.fullname" -}}
{{- if .Values.fullnameOverride }}{{ .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}{{- else }}{{ printf "%s-%s" .Release.Name (default .Chart.Name .Values.nameOverride) | trunc 63 | trimSuffix "-" }}{{- end }}
{{- end }}
{{- define "trussium-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- define "trussium-operator.labels" -}}
helm.sh/chart: {{ include "trussium-operator.chart" . }}
{{ include "trussium-operator.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: operator
{{- end }}
{{- define "trussium-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "trussium-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
{{- define "trussium-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}{{ default (include "trussium-operator.fullname" .) .Values.serviceAccount.name }}{{- else }}{{ default "default" .Values.serviceAccount.name }}{{- end }}
{{- end }}
{{- define "trussium-operator.image" -}}
{{- printf "%s:%s" .Values.image.repository (default .Chart.AppVersion .Values.image.tag) }}
{{- end }}
