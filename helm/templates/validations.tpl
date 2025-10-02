{{- /*
Validations which are not possible to integrate into the JSON schema
due to limitations in Helm's schema validation capabilities and/or
limitations in JSON Schema.
*/ -}}

{{- /* Certificate algorithm-specific validations for cross-field dependencies */ -}}
{{- with .Values.insightsController.tls.certificate -}}
  {{- if eq .algorithm "rsa" -}}
    {{- if not .keySize -}}
      {{- fail "RSA algorithm requires keySize to be specified" -}}
    {{- end -}}
  {{- end -}}
  {{- if eq .algorithm "ed25519" -}}
    {{- if and .keySize (ne (.keySize | int) 0) -}}
      {{- fail "Ed25519 does not support custom key sizes, omit keySize or use keySize: 0" -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
