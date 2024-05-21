{{/*
Expand the name of the chart.
*/}}
{{- define "cloudzero-agent.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "cloudzero-agent.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Define the secret name which holds the CloudZero API key */}}
{{ define "cloudzero-agent.secretName" -}}
{{ .Values.existingSecretName | default (printf "%s-api-key" .Release.Name) }}
{{- end}}

{{ define "cloudzero-agent.configMapName" -}}
{{ .Values.prometheusConfig.configMapNameOverride | default (printf "%s-configuration" .Release.Name) }}
{{- end}}

{{/*
Create labels for prometheus
*/}}
{{- define "cloudzero-agent.common.matchLabels" -}}
app.kubernetes.io/name: {{ include "cloudzero-agent.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "cloudzero-agent.server.matchLabels" -}}
app.kubernetes.io/component: {{ .Values.server.name }}
{{ include "cloudzero-agent.common.matchLabels" . }}
{{- end -}}

{{/*
Create unified labels for prometheus components
*/}}
{{- define "cloudzero-agent.common.metaLabels" -}}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
helm.sh/chart: {{ include "cloudzero-agent.chart" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: {{ include "cloudzero-agent.name" . }}
{{- with .Values.commonMetaLabels}}
{{ toYaml . }}
{{- end }}
{{- end -}}

{{- define "cloudzero-agent.server.labels" -}}
{{ include "cloudzero-agent.server.matchLabels" . }}
{{ include "cloudzero-agent.common.metaLabels" . }}
{{- end -}}


{{/*
Define the cloudzero-agent.namespace template if set with forceNamespace or .Release.Namespace is set
*/}}
{{- define "cloudzero-agent.namespace" -}}
  {{- default .Release.Namespace .Values.forceNamespace -}}
{{- end }}

{{/*
Create the name of the service account to use for the server component
*/}}
{{- define "cloudzero-agent.serviceAccountName.server" -}}
{{- if .Values.serviceAccounts.server.create -}}
    {{ default (include "cloudzero-agent.server.fullname" .) .Values.serviceAccounts.server.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccounts.server.name }}
{{- end -}}
{{- end -}}

{{/*
Create a fully qualified Prometheus server name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "cloudzero-agent.server.fullname" -}}
{{- if .Values.server.fullnameOverride -}}
{{- .Values.server.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- printf "%s-%s" .Release.Name .Values.server.name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s-%s" .Release.Name $name "server" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Return the appropriate apiVersion for rbac.
*/}}
{{- define "cloudzero-agent.rbac.apiVersion" -}}
{{- if .Capabilities.APIVersions.Has "rbac.authorization.k8s.io/v1" }}
{{- print "rbac.authorization.k8s.io/v1" -}}
{{- else -}}
{{- print "rbac.authorization.k8s.io/v1beta1" -}}
{{- end -}}
{{- end -}}

{{/*
Create a fully qualified ClusterRole name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "cloudzero-agent.clusterRoleName" -}}
{{- if .Values.server.clusterRoleNameOverride -}}
{{ .Values.server.clusterRoleNameOverride | trunc 63 | trimSuffix "-" }}
{{- else -}}
{{ include "cloudzero-agent.server.fullname" . }}
{{- end -}}
{{- end -}}

{{/*
Combine metric lists
*/}}
{{- define "chartname.combineMetrics" -}}
{{- $total := append .Values.kubeMetrics .Values.containerMetrics -}}
{{- $result := join "|" $total -}}
{{- $result -}}
{{- end -}}
