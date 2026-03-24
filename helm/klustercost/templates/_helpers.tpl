{{/*
Common labels applied to every resource in the chart.
*/}}
{{- define "klustercost.labels" -}}
app.kubernetes.io/name: {{ .Values.global.appName | default "klustercost" }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{- with .Values.global.extraLabels }}
{{ toYaml . | trim }}
{{- end }}
{{- end -}}

{{/*
Selector labels — stable subset used for matchLabels.
Must not change after initial deploy.
*/}}
{{- define "klustercost.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.global.appName | default "klustercost" }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Component labels — includes common labels plus component identifier.
Usage: include "klustercost.componentLabels" (dict "context" . "component" "monitor")
*/}}
{{- define "klustercost.componentLabels" -}}
{{ include "klustercost.labels" .context }}
app.kubernetes.io/component: {{ .component }}
{{- end -}}

{{/*
Component selector labels.
Usage: include "klustercost.componentSelectorLabels" (dict "context" . "component" "monitor")
*/}}
{{- define "klustercost.componentSelectorLabels" -}}
{{ include "klustercost.selectorLabels" .context }}
app.kubernetes.io/component: {{ .component }}
{{- end -}}
