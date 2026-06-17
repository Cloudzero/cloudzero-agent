{{/*
Webhook server enablement resolver.

Single source of truth for whether the admission webhook server (née Insights
Controller) and its supporting resources should be rendered.

Precedence:
  1. Legacy insightsController.enabled, when explicitly set (a bool), wins.
     (Its default is null, so an explicit true/false is distinguishable.)
  2. Otherwise components.webhookServer.enabled (true | false | "auto").
     "auto" and unset both currently resolve to enabled (true). A future
     release will make "auto" follow the KubeState configuration; that is the
     only line to change here when promoting that behavior.

Returns "true" (truthy) when enabled, or "" (empty/falsy) when disabled —
the standard truthy-string / empty-string idiom used by the chart's other
boolean-resolving helpers.

Usage: {{ if eq (include "cloudzero-agent.webhookServer.enabled" .) "true" }}...{{ end }}
*/}}
{{- define "cloudzero-agent.webhookServer.enabled" -}}
{{- $ic := .Values.insightsController.enabled -}}
{{- if not (kindIs "invalid" $ic) -}}
  {{- /* Legacy precedence: explicitly set insightsController.enabled wins. */ -}}
  {{- if $ic -}}{{- true -}}{{- end -}}
{{- else -}}
  {{- $w := .Values.components.webhookServer.enabled -}}
  {{- if kindIs "invalid" $w -}}
    {{- /* unset = auto */ -}}
    {{- true -}}
  {{- else if eq (toString $w) "auto" -}}
    {{- /* auto = true for now; future: follow KubeState */ -}}
    {{- true -}}
  {{- else if eq (toString $w) "true" -}}
    {{- true -}}
  {{- end -}}
{{- end -}}
{{- end -}}
