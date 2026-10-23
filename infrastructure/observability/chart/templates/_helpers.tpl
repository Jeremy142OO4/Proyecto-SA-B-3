{{/* Render the existing dashboards for the selected application without changing
their UIDs, data source UIDs, users, storage or isolated scenario environment. */}}
{{- define "bank.profile" -}}
{{- $content := .content | replace "bank-usac-dev" .root.Values.applicationNamespace -}}
{{- $content = $content | replace "\\\"dev\\\"" (printf "\\\"%s\\\"" .root.Values.environment) -}}
{{- $content = $content | replace "\"dev\"" (.root.Values.environment | toJson) -}}
{{- if eq .root.Values.environment "prod" -}}
{{- $content = $content | replace "Observabilidad dev" "Observabilidad producción" | replace "Desarrollo" "Producción" -}}
{{- end -}}
{{- $content -}}
{{- end -}}
