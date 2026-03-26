{{- define "fincore.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "fincore.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s" (include "fincore.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "fincore.serviceName" -}}
{{- printf "%s-%s" (include "fincore.fullname" .root) .svcName | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "fincore.configMapName" -}}
{{- printf "%s-config" (include "fincore.serviceName" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "fincore.image" -}}
{{- printf "%s/%s/%s:%s" .root.Values.image.registry .root.Values.image.repository .svcName .root.Values.image.tag -}}
{{- end -}}
