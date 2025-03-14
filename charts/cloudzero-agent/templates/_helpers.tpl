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

{{/* Define the path and filename on the container filesystem which holds the CloudZero API key */}}
{{ define "cloudzero-agent.secretFileFullPath" -}}
{{ printf "%s%s" .Values.serverConfig.containerSecretFilePath .Values.serverConfig.containerSecretFileName }}
{{- end}}

{{/*
imagePullSecrets for the agent server
*/}}
{{- define "cloudzero-agent.server.imagePullSecrets" -}}
{{- if .Values.imagePullSecrets -}}
imagePullSecrets:
{{ toYaml .Values.imagePullSecrets | indent 2 -}}
{{- end }}
{{- end }}

{{/*
Name for the validating webhook
*/}}
{{- define "cloudzero-agent.validatingWebhookName" -}}
{{- printf "%s.%s.svc" (include "cloudzero-agent.validatingWebhookConfigName" .) .Release.Namespace }}
{{- end }}

{{ define "cloudzero-agent.configMapName" -}}
{{ .Values.configMapNameOverride | default (printf "%s-configuration" .Release.Name) }}
{{- end}}

{{ define "cloudzero-agent.validatorConfigMapName" -}}
{{- printf "%s-validator-configuration" .Release.Name -}}
{{- end}}

{{/*
  This helper function trims whitespace and newlines from a given string.
*/}}
{{- define "cloudzero-agent.cleanString" -}}
  {{- $input := . -}}
  {{- $cleaned := trimAll "\n\t\r\f\v ~`!@#$%^&*()[]{}_-+=|\\:;\"'<,>.?/" $input -}}
  {{- $cleaned := trim $cleaned -}}
  {{- $cleaned -}}
{{- end -}}

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
{{- define "cloudzero-agent.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (include "cloudzero-agent.server.fullname" .) .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.server.serviceAccount.name }}
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
{{- define "cloudzero-agent.combineMetrics" -}}
{{- $total := concat .Values.kubeMetrics .Values.containerMetrics .Values.insightsMetrics .Values.prometheusMetrics -}}
{{- $result := join "|" $total -}}
{{- $result -}}
{{- end -}}

{{/*
Generate metric filters
*/}}
{{- define "cloudzero-agent.generateMetricFilters" -}}
{{- if ne 0 (add (len .filters.exact) (len .filters.additionalExact) (len .filters.prefix) (len .filters.additionalPrefix) (len .filters.suffix) (len .filters.additionalSuffix) (len .filters.contains) (len .filters.additionalContains) (len .filters.regex) (len .filters.additionalRegex)) }}
{{ .name }}:
{{- range $pattern := uniq (concat .filters.exact .filters.additionalExact) }}
  - pattern: "{{ $pattern }}"
    match: exact
{{- end }}
{{- range $pattern := uniq (concat .filters.prefix .filters.additionalPrefix) }}
  - pattern: "{{ $pattern }}"
    match: prefix
{{- end }}
{{- range $pattern := uniq (concat .filters.suffix .filters.additionalSuffix) }}
  - pattern: "{{ $pattern }}"
    match: suffix
{{- end }}
{{- range $pattern := uniq (concat .filters.contains .filters.additionalContains) }}
  - pattern: "{{ $pattern }}"
    match: contains
{{- end }}
{{- range $pattern := uniq (concat .filters.regex .filters.additionalRegex) }}
  - pattern: "{{ $pattern }}"
    match: regex
{{- end }}
{{- end -}}
{{- end -}}

{{/*
Required metric labels
*/}}
{{- define "cloudzero-agent.requiredMetricLabels" -}}
{{- $requiredSpecialMetricLabels := tuple "_.*" "label_.*" "app.kubernetes.io/*" "k8s.*" -}}
{{- $requiredCZMetricLabels := tuple "board_asset_tag" "container" "created_by_kind" "created_by_name" "image" "instance" "name" "namespace" "node" "node_kubernetes_io_instance_type" "pod" "product_name" "provider_id" "resource" "unit" "uid" -}}
{{- $total := concat .Values.additionalMetricLabels $requiredCZMetricLabels $requiredSpecialMetricLabels -}}
{{- $result := join "|" $total -}}
{{- $result -}}
{{- end -}}

{{/*
The name of the KSM service target that will be used in the scrape config and validator
*/}}
{{- define "cloudzero-agent.kubeStateMetrics.kubeStateMetricsSvcTargetName" -}}
{{- $name := "" -}}
{{/* If the user specifies an override for the service name, use it. */}}
{{- if .Values.kubeStateMetrics.targetOverride -}}
{{ .Values.kubeStateMetrics.targetOverride }}
{{/* After the first override option is not used, try to mirror what the KSM chart does internally. */}}
{{- else if .Values.kubeStateMetrics.fullnameOverride -}}
{{- $svcName := .Values.kubeStateMetrics.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{ printf "%s.%s.svc.cluster.local:%d" $svcName .Release.Namespace (int .Values.kubeStateMetrics.service.port) | trim }}
{{/* If KSM is not enabled, and they haven't set a targetOverride, fail the installation */}}
{{- else if not .Values.kubeStateMetrics.enabled -}}
{{- required "You must set a targetOverride for kubeStateMetrics" .Values.kubeStateMetrics.targetOverride -}}
{{/* This is the case where the user has not tried to change the name and are still using the internal KSM */}}
{{- else if .Values.kubeStateMetrics.enabled -}}
{{- $name = default .Chart.Name .Values.kubeStateMetrics.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- $svcName := .Release.Name | trunc 63 | trimSuffix "-" -}}
{{ printf "%s.%s.svc.cluster.local:%d" $svcName .Release.Namespace (int .Values.kubeStateMetrics.service.port) | trim }}
{{- else -}}
{{- $svcName := printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{ printf "%s.%s.svc.cluster.local:%d" $svcName .Release.Namespace (int .Values.kubeStateMetrics.service.port) | trim }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Insights Controller
*/}}

{{/*
Create common matchLabels for webhook server
*/}}
{{- define "cloudzero-agent.insightsController.common.matchLabels" -}}
app.kubernetes.io/name: {{ include "cloudzero-agent.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "cloudzero-agent.insightsController.server.matchLabels" -}}
app.kubernetes.io/component: {{ .Values.insightsController.server.name }}
{{ include "cloudzero-agent.common.matchLabels" . }}
{{- end -}}

{{- define "cloudzero-agent.insightsController.initBackfillJob.matchLabels" -}}
app.kubernetes.io/component: {{ include "cloudzero-agent.initBackfillJobName" . }}
{{ include "cloudzero-agent.common.matchLabels" . }}
{{- end -}}

{{- define "cloudzero-agent.insightsController.initCertJob.matchLabels" -}}
app.kubernetes.io/component: {{ include "cloudzero-agent.initCertJobName" . }}
{{ include "cloudzero-agent.common.matchLabels" . }}
{{- end -}}

{{/*
Create common matchLabels for aggregator
*/}}
{{- define "cloudzero-agent.aggregator.matchLabels" -}}
app.kubernetes.io/component: aggregator
app.kubernetes.io/name: {{ include "cloudzero-agent.aggregator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
imagePullSecrets for the insights controller webhook server
*/}}
{{- define "cloudzero-agent.insightsController.server.imagePullSecrets" -}}
{{- if .Values.insightsController.server.imagePullSecrets -}}
imagePullSecrets:
{{ toYaml .Values.insightsController.server.imagePullSecrets | indent 2 }}
{{- else if .Values.imagePullSecrets }}
imagePullSecrets:
{{ toYaml .Values.imagePullSecrets | indent 2 }}
{{- end }}
{{- end }}

{{/*
imagePullSecrets for the insights controller init scrape job.
Defaults to given value, then the insightsController value, then the top level value
*/}}
{{- define "cloudzero-agent.initBackfillJob.imagePullSecrets" -}}
{{ $backFillValues := (include "cloudzero-agent.backFill" . | fromYaml) }}
{{- if $backFillValues.imagePullSecrets -}}
imagePullSecrets:
{{ toYaml $backFillValues.imagePullSecrets | indent 2 }}
{{- else if .Values.insightsController.server.imagePullSecrets -}}
imagePullSecrets:
{{ toYaml .Values.insightsController.server.imagePullSecrets | indent 2 }}
{{- else if .Values.imagePullSecrets }}
imagePullSecrets:
{{ toYaml .Values.imagePullSecrets | indent 2 }}
{{- end }}
{{- end }}

{{/*
imagePullSecrets for the insights controller init cert job.
Defaults to given value, then the insightsController value, then the top level value
*/}}
{{- define "cloudzero-agent.initCertJob.imagePullSecrets" -}}
{{- if .Values.initCertJob.imagePullSecrets -}}
imagePullSecrets:
{{ toYaml .Values.initCertJob.imagePullSecrets | indent 2 }}
{{- else if .Values.insightsController.server.imagePullSecrets -}}
imagePullSecrets:
{{ toYaml .Values.insightsController.server.imagePullSecrets | indent 2 }}
{{- else if .Values.imagePullSecrets -}}
imagePullSecrets:
{{ toYaml .Values.imagePullSecrets | indent 2 }}
{{- end }}
{{- end }}


{{/*
Get the full container image reference for the init scrape job pod
*/}}
{{- define "cloudzero-agent.initBackfillJob.imageReference" -}}
{{ $backFillValues := (include "cloudzero-agent.backFill" .) | fromYaml }}
{{- $repository := .Values.insightsController.server.image.repository -}}
{{ $tag := .Values.insightsController.server.image.tag -}}
{{- if and $backFillValues.image $backFillValues.image.repository -}}
{{- $repository = $backFillValues.image.repository }}
{{- end }}
{{- if and $backFillValues.image $backFillValues.image.tag -}}
{{- $tag = $backFillValues.image.tag -}}
{{- end }}
{{- printf "%s:%s" $repository $tag }}
{{- end }}

{{/*
Service selector labels
*/}}
{{- define "cloudzero-agent.selectorLabels" -}}
{{ include "cloudzero-agent.common.matchLabels" . }}
{{ include "cloudzero-agent.insightsController.server.matchLabels" . }}
{{- end }}

{{- define "cloudzero-agent.insightsController.labels" -}}
{{ include "cloudzero-agent.insightsController.server.matchLabels" . }}
{{ include "cloudzero-agent.common.metaLabels" . }}
{{- end -}}

{{- define "cloudzero-agent.aggregator.selectorLabels" -}}
{{ include "cloudzero-agent.common.matchLabels" . }}
{{ include "cloudzero-agent.aggregator.matchLabels" . }}
{{- end }}

{{- define "cloudzero-agent.aggregator.labels" -}}
{{ include "cloudzero-agent.aggregator.matchLabels" . }}
{{ include "cloudzero-agent.common.metaLabels" . }}
{{- end -}}

{{/*
Create a fully qualified webhook server name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "cloudzero-agent.insightsController.server.webhookFullname" -}}
{{- if .Values.server.fullnameOverride -}}
{{- .Values.server.fullnameOverride | trunc 63 | trimSuffix "-" -}}-webhook
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- printf "%s-%s" .Release.Name .Values.insightsController.server.name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s-%s" .Release.Name $name .Values.insightsController.server.name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Name for the webhook server service
*/}}
{{- define "cloudzero-agent.serviceName" -}}
{{- printf "%s-svc" (include "cloudzero-agent.insightsController.server.webhookFullname" .) }}
{{- end }}

{{/*
Name for the validating webhook configuration resource
*/}}
{{- define "cloudzero-agent.validatingWebhookConfigName" -}}
{{- printf "%s-webhook" (include "cloudzero-agent.insightsController.server.webhookFullname" .) }}
{{- end }}


{{ define "cloudzero-agent.webhookConfigMapName" -}}
{{ .Values.insightsController.ConfigMapNameOverride | default (printf "%s-webhook-configuration" .Release.Name) }}
{{- end}}

{{ define "cloudzero-agent.aggregator.name" -}}
{{ .Values.aggregator.name | default (printf "%s-aggregator" .Release.Name) }}
{{- end}}

{{/*
Mount path for the insights server configuration file
*/}}
{{- define "cloudzero-agent.insightsController.configurationMountPath" -}}
{{- default .Values.insightsController.configurationMountPath (printf "/etc/%s-insights" .Chart.Name)  }}
{{- end }}

{{/*
Name for the issuer resource
*/}}
{{- define "cloudzero-agent.issuerName" -}}
{{- printf "%s-issuer" (include "cloudzero-agent.insightsController.server.webhookFullname" .) }}
{{- end }}

{{/*
Map for initBackfillJob values; this allows us to preferably use initBackfillJob, but if users are still using the deprecated initScrapeJob, we will accept those as well
*/}}
{{- define "cloudzero-agent.backFill" -}}
{{- merge .Values.initBackfillJob (.Values.initScrapeJob | default (dict)) | toYaml }}
{{- end }}

{{/*
Name for the backfill job resource
*/}}
{{- define "cloudzero-agent.initBackfillJobName" -}}
{{- $name := printf "%s-backfill-%s" .Release.Name .Chart.Version }}
{{- $imageRef := splitList ":" (include  "cloudzero-agent.initBackfillJob.imageReference" .) | last }}
{{- printf "%s-%s" $name ($imageRef | trunc 8) | trunc 63 | replace "." "-" }}
{{- end }}

{{/*
Name for the certificate init job resource
*/}}
{{- define "cloudzero-agent.initCertJobName" -}}
{{- printf "%s-init-cert" (include "cloudzero-agent.insightsController.server.webhookFullname" .) }}
{{- end }}

{{/*
Annotations for the webhooks
*/}}
{{- define "cloudzero-agent.webhooks.annotations" -}}
{{- if or .Values.insightsController.tls.useCertManager .Values.insightsController.webhooks.annotations }}
annotations:
{{- if .Values.insightsController.webhooks.annotations }}
{{ toYaml .Values.insightsController.webhook.annotations | nindent 2}}
{{- end }}
{{- if .Values.insightsController.tls.useCertManager }}
  cert-manager.io/inject-ca-from: {{ .Values.insightsController.webhooks.caInjection | default (printf "%s/%s" .Release.Namespace (include "cloudzero-agent.certificateName" .)) }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Name for the certificate resource
*/}}
{{- define "cloudzero-agent.certificateName" -}}
{{- printf "%s-certificate" (include "cloudzero-agent.insightsController.server.webhookFullname" .) }}
{{- end }}

{{/*
Name for the secret holding TLS certificates
*/}}
{{- define "cloudzero-agent.tlsSecretName" -}}
{{- .Values.insightsController.tls.secret.name | default (printf "%s-tls" (include "cloudzero-agent.insightsController.server.webhookFullname" .)) }}
{{- end }}

{{/*
Volume mount for the API key
*/}}
{{- define "cloudzero-agent.apiKeyVolumeMount" -}}
{{- if or .Values.existingSecretName .Values.apiKey -}}
- name: cloudzero-api-key
  mountPath: {{ .Values.serverConfig.containerSecretFilePath }}
  subPath: ""
  readOnly: true
{{- end }}
{{- end }}
