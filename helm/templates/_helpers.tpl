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
{{- $labels := dict
    "app.kubernetes.io/version" .Chart.AppVersion
    "helm.sh/chart" (include "cloudzero-agent.chart" .)
    "app.kubernetes.io/managed-by" .Release.Service
    "app.kubernetes.io/part-of" (include "cloudzero-agent.name" .)
-}}
{{- (merge $labels .Values.defaults.labels .Values.commonMetaLabels) | toYaml -}}
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
Create the name of the service account to use for the init-cert Job
*/}}
{{- define "cloudzero-agent.initCertJob.serviceAccountName" -}}
{{- $defaultName := (printf "%s-init-cert" (include "cloudzero-agent.insightsController.server.webhookFullname" .)) | trunc 63 -}}
{{ .Values.initCertJob.rbac.serviceAccountName | default $defaultName }}
{{- end -}}

{{/*
Create the name of the ClusterRole to use for the init-cert Job
*/}}
{{- define "cloudzero-agent.initCertJob.clusterRoleName" -}}
{{- $defaultName := (printf "%s-init-cert" (include "cloudzero-agent.insightsController.server.webhookFullname" .)) | trunc 63 -}}
{{ .Values.initCertJob.rbac.clusterRoleName | default $defaultName }}
{{- end -}}

{{/*
Create the name of the ClusterRoleBinding to use for the init-cert Job
*/}}
{{- define "cloudzero-agent.initCertJob.clusterRoleBindingName" -}}
{{- $defaultName := (printf "%s-init-cert" (include "cloudzero-agent.insightsController.server.webhookFullname" .)) | trunc 63 -}}
{{ .Values.initCertJob.rbac.clusterRoleBinding | default $defaultName }}
{{- end -}}

{{/*
init-cert Job annotations
*/}}
{{- define "cloudzero-agent.initCertJob.annotations" -}}
{{- if .Values.initCertJob.annotations -}}
annotations:
  {{- toYaml .Values.initCertJob.annotations | nindent 2 -}}
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
{{- $total := concat (include "cloudzero-agent.defaults" . | fromYaml).kubeMetrics (include "cloudzero-agent.defaults" . | fromYaml).containerMetrics (include "cloudzero-agent.defaults" . | fromYaml).insightsMetrics (include "cloudzero-agent.defaults" . | fromYaml).prometheusMetrics -}}
{{- $result := join "|" $total -}}
{{- $result -}}
{{- end -}}

{{/*
Internal helper function for generating a metric filter regex
*/}}
{{- define "cloudzero-agent.generateMetricFilterRegexInternal" -}}
{{- $patterns := list -}}
{{/* Handle exact matches */}}
{{- $exactPatterns := uniq .exact -}}
{{- if gt (len $exactPatterns) 0 -}}
{{- $exactPattern := printf "^(%s)$" (join "|" $exactPatterns) -}}
{{- $patterns = append $patterns $exactPattern -}}
{{- end -}}

{{/* Handle prefix matches */}}
{{- $prefixPatterns := uniq .prefix -}}
{{- if gt (len $prefixPatterns) 0 -}}
{{- $prefixPattern := printf "^(%s)" (join "|" $prefixPatterns) -}}
{{- $patterns = append $patterns $prefixPattern -}}
{{- end -}}

{{/* Handle suffix matches */}}
{{- $suffixPatterns := uniq .suffix -}}
{{- if gt (len $suffixPatterns) 0 -}}
{{- $suffixPattern := printf "(%s)$" (join "|" $suffixPatterns) -}}
{{- $patterns = append $patterns $suffixPattern -}}
{{- end -}}

{{/* Handle contains matches */}}
{{- $containsPatterns := uniq .contains -}}
{{- if gt (len $containsPatterns) 0 -}}
{{- $containsPattern := printf "(%s)" (join "|" $containsPatterns) -}}
{{- $patterns = append $patterns $containsPattern -}}
{{- end -}}

{{/* Handle regex matches */}}
{{- $regexPatterns := uniq .regex -}}
{{- if gt (len $regexPatterns) 0 -}}
{{- $regexPattern := printf "(%s)" (join "|" $regexPatterns) -}}
{{- $patterns = append $patterns $regexPattern -}}
{{- end -}}

{{- join "|" $patterns -}}
{{- end -}}

{{- define "cloudzero-agent.generateMetricNameFilterRegex" -}}
{{- include "cloudzero-agent.generateMetricFilterRegexInternal" (dict
  "exact"    (uniq (concat (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.cost.name.exact    (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.observability.name.exact   ))
  "prefix"   (uniq (concat (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.cost.name.prefix   (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.observability.name.prefix  ))
  "suffix"   (uniq (concat (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.cost.name.suffix   (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.observability.name.suffix  ))
  "contains" (uniq (concat (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.cost.name.contains (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.observability.name.contains))
  "regex"    (uniq (concat (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.cost.name.regex    (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.observability.name.regex   ))
) -}}
{{- end -}}

{{- define "cloudzero-agent.generateMetricLabelFilterRegex" -}}
{{- include "cloudzero-agent.generateMetricFilterRegexInternal" (dict
  "exact"    (uniq (concat (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.cost.labels.exact    (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.observability.labels.exact   ))
  "prefix"   (uniq (concat (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.cost.labels.prefix   (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.observability.labels.prefix  ))
  "suffix"   (uniq (concat (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.cost.labels.suffix   (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.observability.labels.suffix  ))
  "contains" (uniq (concat (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.cost.labels.contains (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.observability.labels.contains))
  "regex"    (uniq (concat (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.cost.labels.regex    (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.observability.labels.regex   ))
) -}}
{{- end -}}

{{/*
Generate metric filters
*/}}
{{- define "cloudzero-agent.generateMetricFilters" -}}
{{- if ne 0 (add (len .filters.exact) (len .filters.prefix) (len .filters.suffix) (len .filters.contains) (len .filters.regex)) }}
{{ .name }}:
{{- range $pattern := uniq .filters.exact }}
  - pattern: "{{ $pattern }}"
    match: exact
{{- end }}
{{- range $pattern := uniq .filters.prefix }}
  - pattern: "{{ $pattern }}"
    match: prefix
{{- end }}
{{- range $pattern := uniq .filters.suffix }}
  - pattern: "{{ $pattern }}"
    match: suffix
{{- end }}
{{- range $pattern := uniq .filters.contains }}
  - pattern: "{{ $pattern }}"
    match: contains
{{- end }}
{{- range $pattern := uniq .filters.regex }}
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
{{- $total := concat $requiredCZMetricLabels $requiredSpecialMetricLabels -}}
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
{{- printf "%s-webhook" .Values.server.fullnameOverride | trunc 63 | trimSuffix "-" -}}
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
Name for the webhook server Deployment
*/}}
{{- define "cloudzero-agent.insightsController.deploymentName" -}}
{{- include "cloudzero-agent.insightsController.server.webhookFullname" . }}
{{- end }}

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
Name for a job resource
*/}}
{{- define "cloudzero-agent.jobName" -}}
{{- printf "%s-%s-%s" .Release .Name (.Values.jobConfigID | default (. | toYaml | sha256sum)) | trunc 61 -}}
{{- end }}

{{/*
Name for the backfill job resource
*/}}
{{- define "cloudzero-agent.initBackfillJobName" -}}
{{- include "cloudzero-agent.jobName" (dict "Release" .Release.Name "Name" "backfill" "Version" .Chart.Version "Values" .Values) -}}
{{- end }}

{{/*
initBackfillJob Job annotations
*/}}
{{- define "cloudzero-agent.initBackfillJob.annotations" -}}
{{- if .Values.initBackfillJob.annotations -}}
annotations:
  {{- toYaml .Values.initBackfillJob.annotations | nindent 2 -}}
{{- end -}}
{{- end -}}

{{/*
Name for the certificate init job resource. Should be a new name each installation/upgrade.
*/}}
{{- define "cloudzero-agent.initCertJobName" -}}
{{- include "cloudzero-agent.jobName" (dict "Release" .Release.Name "Name" "init-cert" "Version" .Chart.Version "Values" .Values) -}}
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

{{/*
Return the URL for the agent and insights controller to send metrics to.

If the CloudZero Aggregator is enabled, this will be the URL for the collector.
Otherwise, it will be the CloudZero API endpoint.

*/}}
{{- define "cloudzero-agent.metricsDestination" -}}
'http://{{ include "cloudzero-agent.aggregator.name" . }}.{{ .Release.Namespace }}.svc.cluster.local/collector'
{{- end -}}

{{/*
Generate image configuration with defaults.
*/}}
{{- define "cloudzero-agent.generateImage" -}}
{{- $digest      := (.image.digest      | default .defaults.digest) -}}
{{- $tag         := (.image.tag         | default .defaults.tag) -}}
{{- $repository  := (.image.repository  | default .defaults.repository) -}}
{{- $pullPolicy  := (.image.pullPolicy  | default .defaults.pullPolicy) -}}
{{- $pullSecrets := (.image.pullSecrets | default .defaults.pullSecrets) -}}
{{- if .compat -}}
{{- $digest      = (.compat.digest      | default .image.digest      | default .defaults.digest) -}}
{{- $tag         = (.compat.tag         | default .image.tag         | default .defaults.tag) -}}
{{- $repository  = (.compat.repository  | default .image.repository  | default .defaults.repository) -}}
{{- $pullPolicy  = (.compat.pullPolicy  | default .image.pullPolicy  | default .defaults.pullPolicy) -}}
{{- $pullSecrets = (.compat.pullSecrets | default .image.pullSecrets | default .defaults.pullSecrets) -}}
{{- end -}}
{{- if $digest -}}
image: "{{ $repository }}@{{ $digest }}"
{{- else if $tag -}}
image: "{{ $repository }}:{{ $tag }}"
{{- end }}
{{ if $pullPolicy -}}
imagePullPolicy: "{{ $pullPolicy }}"
{{- end }}
{{ if $pullSecrets -}}
imagePullSecrets:
{{ toYaml $pullSecrets | indent 2 }}
{{- end }}
{{- end -}}

{{/* Generate priority class name */}}
{{- define "cloudzero-agent.generatePriorityClassName" -}}
{{- if . -}}
priorityClassName: {{ . }}
{{- end -}}
{{- end -}}

{{/* Generate DNS info */}}
{{- define "cloudzero-agent.generateDNSInfo" -}}
{{- $dnsPolicy := .defaults.policy -}}
{{- $dnsConfig := .defaults.config -}}
{{- if $dnsPolicy -}}
dnsPolicy: {{ $dnsPolicy }}
{{- end -}}
{{- if $dnsConfig }}
dnsConfig:
{{ $dnsConfig | toYaml | indent 2 }}
{{ end -}}
{{- end -}}

{{/*
Generate labels for a component
*/}}
{{- define "cloudzero-agent.generateLabels" -}}
{{- $labels := dict
    "app.kubernetes.io/version" .globals.Chart.AppVersion
    "helm.sh/chart" (include "cloudzero-agent.chart" .globals)
    "app.kubernetes.io/managed-by" .globals.Release.Service
    "app.kubernetes.io/part-of" (include "cloudzero-agent.name" .globals)
-}}
{{- if .component -}}
{{- $labels = merge (dict "app.kubernetes.io/component" .component) $labels -}}
{{- end -}}
{{- if len $labels -}}
labels:
{{- (merge $labels (.labels | default (dict)) .globals.Values.defaults.labels .globals.Values.commonMetaLabels) | toYaml | nindent 2 -}}
{{- end -}}
{{- end -}}

{{/*
Generate annotations
*/}}
{{- define "cloudzero-agent.generateAnnotations" -}}
{{- if . -}}
annotations:
{{- . | toYaml | nindent 2 -}}
{{- end -}}
{{- end -}}

{{/*
Generate affinity sections
*/}}
{{- define "cloudzero-agent.generateAffinity" -}}
{{ $affinity := .default }}
{{- if .affinity -}}
{{ $affinity = merge .affinity .default }}
{{- end -}}
{{- if $affinity -}}
affinity:
{{- $affinity | toYaml | nindent 2 -}}
{{- end -}}
{{- end -}}

{{/*
Generate tolerations sections
*/}}
{{- define "cloudzero-agent.generateTolerations" -}}
{{- if . -}}
tolerations:
{{- . | toYaml | nindent 2 }}
{{- end -}}
{{- end -}}

{{/*
Generate nodeSelector sections
*/}}
{{- define "cloudzero-agent.generateNodeSelector" -}}
{{- $nodeSelector := .nodeSelector | default .default -}}
{{if $nodeSelector }}
nodeSelector:
{{- $nodeSelector | toYaml | nindent 2 -}}
{{- end -}}
{{- end -}}

{{/*
Generate a pod disruption budget
*/}}
{{- define "cloudzero-agent.generatePodDisruptionBudget" -}}
{{- $replicas := int (.replicas | default .component.replicas | default 99999) -}}
{{- if (.component.podDisruptionBudget.minAvailable | default .component.podDisruptionBudget.maxUnavailable) }}
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: {{ .name }}
  namespace: {{ .root.Release.Namespace }}
spec:
  {{- if .component.podDisruptionBudget.minAvailable }}
  {{- if le $replicas (int .component.podDisruptionBudget.minAvailable) -}}
  {{- fail (printf "Insufficient replicas in %s (%d) for pod disruption budget minAvailable (%v)" .name $replicas .component.podDisruptionBudget.minAvailable) -}}
  {{- end }}
  minAvailable: {{ .component.podDisruptionBudget.minAvailable }}
  {{- end }}
  {{- if .component.podDisruptionBudget.maxUnavailable }}
  maxUnavailable: {{ .component.podDisruptionBudget.maxUnavailable }}
  {{- end }}
  selector:
    matchLabels:
      {{- .matchLabels | nindent 6 }}
{{- end }}
{{- end -}}
