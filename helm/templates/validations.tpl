{{- /*
Validations which are not possible to integrate into the JSON schema
due to limitations in Helm's schema validation capabilities and/or
limitations in JSON Schema.
*/ -}}

{{- /* You must set either apiKey or existingSecretName. */ -}}
{{- if and (not .Values.apiKey) (not .Values.existingSecretName) }}
  {{- fail "Either apiKey or existingSecretName must be set" -}}
{{- end -}}
