{{/*
Expand the name of the chart.
*/}}
{{- define "api7-ingress-controller-manager.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "api7-ingress-controller-manager.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}
{{/*
Common labels
*/}}
{{- define "api7-ingress-controller-manager.labels" -}}
helm.sh/chart: {{ include "api7-ingress-controller-manager.chart" . }}
{{ include "api7-ingress-controller-manager.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "api7-ingress-controller-manager.selectorLabels" -}}
{{- if .Values.labelsOverride }}
{{- tpl (.Values.labelsOverride | toYaml) . }}
{{- else }}
app.kubernetes.io/name: {{ include "api7-ingress-controller-manager.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
{{- end }}
