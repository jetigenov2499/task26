{{- define "api-chart.name" -}}
{{- default "api" .Chart.Name }}
{{- end }}

{{- define "api-chart.fullname" -}}
{{- printf "%s-%s" .Release.Name (include "api-chart.name" .) }}
{{- end }}