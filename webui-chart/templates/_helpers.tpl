{{- define "webui-chart.name" -}}
{{- default "web-ui" .Chart.Name }}
{{- end }}

{{- define "webui-chart.fullname" -}}
{{- printf "%s-%s" .Release.Name (include "webui-chart.name" .) }}
{{- end }}
