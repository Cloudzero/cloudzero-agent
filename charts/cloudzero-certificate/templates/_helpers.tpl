{{/*
Expand the name of the chart.
*/}}
{{- define "cloudzero-certificate.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "cloudzero-certificate.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "cloudzero-certificate.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "cloudzero-certificate.labels" -}}
helm.sh/chart: {{ include "cloudzero-certificate.chart" . }}
{{ include "cloudzero-certificate.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "cloudzero-certificate.selectorLabels" -}}
app.kubernetes.io/name: {{ include "cloudzero-certificate.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the secret to use
*/}}
{{- define "cloudzero-certificate.secretName" -}}
{{- default (include "cloudzero-certificate.fullname" .) .Values.secret.name }}
{{- end }}

{{/*
Generate certificate for the webhook server
*/}}
{{- define "cloudzero-certificate.genCerts" -}}
{{- $releaseName := required "`cloudzeroAgentReleaseName` must be supplied. This value should be the name of the cloudzero-agent helm release that will be created" .Values.cloudzeroAgentReleaseName -}}
{{- $dnsName :=  printf "%s-svc.%s.svc.cluster.local" $releaseName $.Release.Namespace -}}
{{- $dnsNameDefault :=  printf "%s-webhook-server-svc.%s.svc.cluster.local" $releaseName $.Release.Namespace -}}
{{- $dnsNameShort :=  printf "%s-webhook-server-svc" $releaseName -}}
{{- $ca := genCA "cloudzero-agent-ca" 365 -}}
{{- $cert := genSignedCert $dnsName nil (list $dnsName $dnsNameDefault $dnsNameShort) 9999999 $ca -}}
ca.crt: {{ $cert.Cert | b64enc }}
tls.crt: {{ $cert.Cert | b64enc }}
tls.key: {{ $cert.Key | b64enc }}
{{- end -}}
