{{/*
Create a default fully qualified app name for child chart.
*/}}
{{- define "mychart.childName" -}}
{{- $childChartName := .childChartName -}}
{{- printf "%s-%s" .Release.Name $childChartName | trunc 63 | trimSuffix "-" -}}
{{- end -}}
