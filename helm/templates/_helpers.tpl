{{/*
Expand the name of the chart.

This template provides the base name for all Kubernetes resources created by the chart.
Uses Values.nameOverride if provided, otherwise defaults to Chart.Name.
Ensures compatibility with Kubernetes naming constraints (63 char limit, no trailing hyphens).

Usage: {{ include "cloudzero-agent.name" . }}
Returns: string (e.g., "cloudzero-agent")
*/}}
{{- define "cloudzero-agent.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
The version number of the chart.

This template embeds the software version that corresponds to the chart version.
Used for resource annotations and compatibility tracking between chart and application versions.

Usage: {{ include "cloudzero-agent.versionNumber" . }}
Returns: string with version annotation
*/}}
{{- define "cloudzero-agent.versionNumber" -}}
version: 1.2.9  # <- Software release corresponding to this chart version.
{{- end -}}

{{/*
Create chart name and version as used by the chart label.

This template generates the standard chart label value combining name and version.
Used in the app.kubernetes.io/version label for all resources to track deployments.
Replaces '+' with '_' for Kubernetes label compatibility.

Usage: {{ include "cloudzero-agent.chart" . }}
Returns: string (e.g., "cloudzero-agent-1.2.7")
*/}}
{{- define "cloudzero-agent.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Define the secret name which holds the CloudZero API key.

This template determines the Kubernetes Secret name containing the CloudZero API key.
Supports using an existing secret via Values.existingSecretName or creates a default name.
Used by all components that need API access for data upload and authentication.

Usage: {{ include "cloudzero-agent.secretName" . }}
Returns: string (e.g., "my-release-api-key" or custom existing secret name)
*/}}
{{ define "cloudzero-agent.secretName" -}}
{{ .Values.existingSecretName | default (printf "%s-api-key" .Release.Name) }}
{{- end}}

{{/*
Define the path and filename on the container filesystem which holds the CloudZero API key.

This template constructs the complete file path where the API key is mounted inside containers.
Combines the mount path with the filename from serverConfig values.
Used by collector and shipper components to read API credentials.

Usage: {{ include "cloudzero-agent.secretFileFullPath" . }}
Returns: string (e.g., "/secrets/api-key")
*/}}
{{ define "cloudzero-agent.secretFileFullPath" -}}
{{ printf "%s%s" .Values.serverConfig.containerSecretFilePath .Values.serverConfig.containerSecretFileName }}
{{- end}}

{{/*
imagePullSecrets for the agent server.

This template generates imagePullSecrets configuration for private container registries.
Only renders the imagePullSecrets section if Values.imagePullSecrets is defined.
Applied to all Deployments and Jobs that need to pull CloudZero Agent images.

Usage: {{ include "cloudzero-agent.server.imagePullSecrets" . }}
Returns: YAML imagePullSecrets section or empty string
*/}}
{{- define "cloudzero-agent.server.imagePullSecrets" -}}
{{- if .Values.imagePullSecrets -}}
imagePullSecrets:
{{ toYaml .Values.imagePullSecrets | indent 2 -}}
{{- end }}
{{- end }}

{{/*
Name for the validating webhook.

This template generates the webhook service DNS name for admission controller registration.
Constructs the FQDN using the webhook config name and release namespace.
Used in ValidatingAdmissionWebhook clientConfig to specify the webhook endpoint.

Usage: {{ include "cloudzero-agent.validatingWebhookName" . }}
Returns: string (e.g., "cloudzero-webhook.default.svc")
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

{{ define "cloudzero-agent.helmlessConfigMapName" -}}
{{- printf "%s-helmless-cm" .Release.Name -}}
{{- end}}

{{ define "cloudzero-agent.configLoaderJobName" -}}
{{- include "cloudzero-agent.jobName" (dict "Release" .Release.Name "Name" "confload" "Version" .Chart.Version "Values" .Values) -}}
{{- end}}

{{ define "cloudzero-agent.validatorEnv" -}}
- name: K8S_NAMESPACE
  value: {{ .Release.Namespace }}
- name: K8S_POD_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.name
- name: ISTIO_AMBIENT_REDIRECTION
  valueFrom:
    fieldRef:
      fieldPath: metadata.annotations['ambient.istio.io/redirection']
- name: ISTIO_TOPOLOGY_CLUSTER
  valueFrom:
    fieldRef:
      fieldPath: metadata.labels['topology.istio.io/cluster']
{{- end}}

{{/*
  This helper function trims whitespace and newlines from a given string.
  Returns empty string if input is nil.
*/}}
{{- define "cloudzero-agent.cleanString" -}}
  {{- $input := . -}}
  {{- if $input -}}
    {{- $cleaned := trimAll "\n\t\r\f\v ~`!@#$%^&*()[]{}_-+=|\\:;\"'<,>.?/" $input -}}
    {{- $cleaned := trim $cleaned -}}
    {{- $cleaned -}}
  {{- else -}}
    {{- "" -}}
  {{- end -}}
{{- end -}}

{{/*
Common match labels for selectors
*/}}
{{- define "cloudzero-agent.common.matchLabels" -}}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "cloudzero-agent.server.matchLabels" -}}
app.kubernetes.io/name: server
{{ include "cloudzero-agent.common.matchLabels" . }}
{{- end -}}

{{/*
Common base labels for all Kubernetes resources
*/}}
{{- define "cloudzero-agent.baseLabels" -}}
{{- dict
    "app.kubernetes.io/instance" .Release.Name
    "app.kubernetes.io/version" .Chart.AppVersion
    "helm.sh/chart" (include "cloudzero-agent.chart" .)
    "app.kubernetes.io/managed-by" .Release.Service
    "app.kubernetes.io/part-of" (include "cloudzero-agent.name" .)
| toYaml -}}
{{- end -}}

{{/*
Create unified labels for prometheus components
*/}}
{{- define "cloudzero-agent.common.metaLabels" -}}
{{ (mergeOverwrite
     (include "cloudzero-agent.baseLabels" . | fromYaml)
     (.Values.defaults.labels | default (dict))
     (.Values.commonMetaLabels | default (dict))
   ) | toYaml }}
{{- end -}}

{{- define "cloudzero-agent.server.labels" -}}
{{ (mergeOverwrite
     (include "cloudzero-agent.baseLabels" . | fromYaml)
     (dict "app.kubernetes.io/component" .Values.server.name)
     (.Values.defaults.labels | default (dict))
     (.Values.commonMetaLabels | default (dict))
   ) | toYaml }}
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
{{- $defaultName := include "cloudzero-agent.internal.resourceName" (dict "context" . "component" "webhook" "subcomponent" "init-cert") -}}
{{ .Values.initCertJob.rbac.serviceAccountName | default $defaultName }}
{{- end -}}

{{/*
Create the name of the ClusterRole to use for the init-cert Job
*/}}
{{- define "cloudzero-agent.initCertJob.clusterRoleName" -}}
{{- $defaultName := include "cloudzero-agent.internal.resourceName" (dict "context" . "component" "webhook" "subcomponent" "init-cert") -}}
{{ .Values.initCertJob.rbac.clusterRoleName | default $defaultName }}
{{- end -}}

{{/*
Create the name of the ClusterRoleBinding to use for the init-cert Job
*/}}
{{- define "cloudzero-agent.initCertJob.clusterRoleBindingName" -}}
{{- $defaultName := include "cloudzero-agent.internal.resourceName" (dict "context" . "component" "webhook" "subcomponent" "init-cert") -}}
{{ .Values.initCertJob.rbac.clusterRoleBindingName | default $defaultName }}
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
Override kube-state-metrics subchart naming to use our centralized naming system.

These templates override the kube-state-metrics.fullname and kube-state-metrics.name
helpers from the subchart, allowing us to integrate the subchart's resources into
our naming convention and ensure all names are valid Kubernetes identifiers.

Respects nameOverride from the subchart's own values (not parent chart values).
When this is called from the subchart, .Values refers to the subchart's values.
*/}}
{{- define "kube-state-metrics.fullname" -}}
{{- $component := .Values.nameOverride | default "ksm" -}}
{{- include "cloudzero-agent.internal.resourceName" (dict "context" . "component" $component) -}}
{{- end -}}

{{/*
Override kube-state-metrics.name to ensure container names are valid.
Container names must be lowercase, so we ensure the name is always lowercase.
*/}}
{{- define "kube-state-metrics.name" -}}
{{- .Values.nameOverride | default "ksm" | lower -}}
{{- end -}}

{{/*
Centralized Resource Name Generation

This helper generates consistent Kubernetes resource names for all CloudZero Agent components.

Usage: {{ include "cloudzero-agent.internal.resourceName" (dict "context" . "component" "server") }}

Parameters:
  - context: The template context (usually .)
  - component: Component identifier (e.g., "server", "webhook", "aggregator")
  - subcomponent: Optional subcomponent identifier (e.g., "svc", "init-cert", "tls")
  - suffix: Optional variable suffix (e.g., checksums)
  - override: Optional name override (takes precedence over standard naming)

Naming pattern: {release-name}-cz-{component}[-{subcomponent}][-{suffix}] (truncated to 63 chars)
*/}}
{{- define "cloudzero-agent.internal.resourceName" -}}
  {{- $component := .component -}}
  {{- $subcomponent := .subcomponent | default "" -}}
  {{- $suffix := .suffix | default "" -}}
  {{- $override := .override | default "" -}}
  {{- if $override -}}
    {{- $name := $override -}}
    {{- if $subcomponent -}}
      {{- $name = printf "%s-%s" $name $subcomponent -}}
    {{- end -}}
    {{- if $suffix -}}
      {{- $name = printf "%s-%s" $name $suffix -}}
    {{- end -}}
    {{- $name | trunc 63 | trimSuffix "-" -}}
  {{- else -}}
    {{- $czPrefix := "cz" -}}
    {{- $minSuffixChars := 4 -}}
    {{- $importantParts := $component -}}
    {{- if $subcomponent -}}
      {{- $importantParts = printf "%s-%s" $importantParts $subcomponent -}}
    {{- end -}}
    {{- $suffixSpace := 0 -}}
    {{- if $suffix -}}
      {{- $actualSuffixLen := len $suffix -}}
      {{- if gt $actualSuffixLen $minSuffixChars -}}
        {{- $suffixSpace = add $minSuffixChars 1 -}}
      {{- else -}}
        {{- $suffixSpace = add $actualSuffixLen 1 -}}
      {{- end -}}
    {{- end -}}
    {{- $fixedSpace := add (add (len $czPrefix) (len $importantParts)) 2 -}}
    {{- $requiredLength := add $fixedSpace $suffixSpace -}}
    {{- $maxReleaseLen := int (sub 63 $requiredLength) -}}
    {{- $releaseName := .context.Release.Name -}}
    {{- if gt (len $releaseName) $maxReleaseLen -}}
      {{- $releaseName = trunc $maxReleaseLen $releaseName | trimSuffix "-" -}}
    {{- end -}}
    {{- $name := printf "%s-%s-%s" $releaseName $czPrefix $importantParts -}}
    {{- if $suffix -}}
      {{- $name = printf "%s-%s" $name $suffix -}}
    {{- end -}}
    {{- $name | trunc 63 | trimSuffix "-" -}}
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
  {{- include "cloudzero-agent.internal.resourceName" (dict "context" . "component" "server") -}}
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
{{- if .Values.kubeStateMetrics.targetOverride -}}
{{- .Values.kubeStateMetrics.targetOverride -}}
{{- else if not .Values.kubeStateMetrics.enabled -}}
{{- required "You must set a targetOverride for kubeStateMetrics" .Values.kubeStateMetrics.targetOverride -}}
{{- else -}}
{{- $svcName := include "kube-state-metrics.fullname" (dict "Values" .Values.kubeStateMetrics "Chart" (dict "Name" "kube-state-metrics") "Release" .Release) -}}
{{- printf "%s.%s.svc.cluster.local:%d" $svcName .Release.Namespace (int .Values.kubeStateMetrics.service.port) -}}
{{- end -}}
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
app.kubernetes.io/name: webhook-server
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

{{- define "cloudzero-agent.insightsController.validatorJob.matchLabels" -}}
app.kubernetes.io/component: {{ include "cloudzero-agent.configLoaderJobName" . }}
{{ include "cloudzero-agent.common.matchLabels" . }}
{{- end -}}

{{/*
Create common matchLabels for aggregator
*/}}
{{- define "cloudzero-agent.aggregator.matchLabels" -}}
app.kubernetes.io/name: aggregator
{{ include "cloudzero-agent.common.matchLabels" . }}
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
{{ (mergeOverwrite
     (include "cloudzero-agent.baseLabels" . | fromYaml)
     (dict "app.kubernetes.io/component" .Values.insightsController.server.name)
     (.Values.defaults.labels | default (dict))
     (.Values.commonMetaLabels | default (dict))
   ) | toYaml }}
{{- end -}}

{{- define "cloudzero-agent.aggregator.selectorLabels" -}}
{{ include "cloudzero-agent.common.matchLabels" . }}
{{ include "cloudzero-agent.aggregator.matchLabels" . }}
{{- end }}

{{- define "cloudzero-agent.aggregator.labels" -}}
{{ (mergeOverwrite
     (include "cloudzero-agent.baseLabels" . | fromYaml)
     (dict "app.kubernetes.io/component" "aggregator")
     (.Values.defaults.labels | default (dict))
     (.Values.commonMetaLabels | default (dict))
     (dict "app.kubernetes.io/name" (include "cloudzero-agent.aggregator.name" .))
   ) | toYaml }}
{{- end -}}

{{/*
Create a fully qualified webhook server name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "cloudzero-agent.insightsController.server.webhookFullname" -}}
{{- include "cloudzero-agent.internal.resourceName" (dict "context" . "component" "webhook") -}}
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
{{- include "cloudzero-agent.internal.resourceName" (dict "context" . "component" "webhook") -}}
{{- end }}

{{/*
Name for the validating webhook configuration resource
*/}}
{{- define "cloudzero-agent.validatingWebhookConfigName" -}}
{{- include "cloudzero-agent.internal.resourceName" (dict "context" . "component" "webhook") -}}
{{- end }}


{{ define "cloudzero-agent.webhookConfigMapName" -}}
{{ .Values.insightsController.ConfigMapNameOverride | default (include "cloudzero-agent.internal.resourceName" (dict "context" . "component" "webhook" "subcomponent" "configuration")) }}
{{- end}}

{{ define "cloudzero-agent.aggregator.name" -}}
{{ include "cloudzero-agent.internal.resourceName" (dict "context" . "component" "aggregator" "override" .Values.aggregator.name) }}
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
{{- include "cloudzero-agent.internal.resourceName" (dict "context" . "component" "webhook" "subcomponent" "issuer") }}
{{- end }}

{{/*
Map for initBackfillJob values; this allows us to preferably use initBackfillJob, but if users are still using the deprecated initScrapeJob, we will accept those as well
*/}}
{{- define "cloudzero-agent.backFill" -}}
{{- merge (deepCopy .Values.initBackfillJob) (.Values.initScrapeJob | default (dict)) | toYaml }}
{{- end }}

{{/*
Name for a job resource
*/}}
{{- define "cloudzero-agent.jobName" -}}
{{- printf "%s-%s-%s" .Release .Name (include "cloudzero-agent.configurationChecksum" .) | trunc 61 -}}
{{- end }}

{{/*
Return a hash of the configuration, unless overridden.

Note that jobConfigID *only* exists so we can avoid lots of commit noise when
regenerating the manifests in tests/helm/template. It should never be set in
production, as it will break important functionality to automatically reload
things when a ConfigMap changes.

When used as a subchart, .global values from the parent chart are excluded
from the checksum.
*/}}
{{- define "cloudzero-agent.configurationChecksum" -}}
{{- if .Values.jobConfigID -}}
{{ .Values.jobConfigID }}
{{- else -}}
{{- $cleanValues := omit .Values "global" -}}
{{- if $cleanValues.kubeStateMetrics -}}
  {{- $cleanKSM := omit $cleanValues.kubeStateMetrics "global" -}}
  {{- $_ := set $cleanValues "kubeStateMetrics" $cleanKSM -}}
{{- end -}}
{{- $context := dict "Chart" .Chart "Release" .Release "Values" $cleanValues "Capabilities" .Capabilities -}}
{{ $context | toYaml | sha256sum }}
{{- end -}}
{{- end -}}

{{/*
Name for the backfill job resource
*/}}
{{- define "cloudzero-agent.initBackfillJobName" -}}
{{- printf "%s-backfill-%s" .Release.Name (include "cloudzero-agent.configurationChecksum" .) | trunc 52 -}}
{{- end }}

{{/*
Name for the backfill cronjob resource (without unique ID since it's persistent)
*/}}
{{- define "cloudzero-agent.initBackfillCronJobName" -}}
{{- printf "%s-backfill" .Release.Name -}}
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
Name for the helmless job resource. Should be a new name each installation/upgrade.
*/}}
{{- define "cloudzero-agent.helmlessJobName" -}}
{{- include "cloudzero-agent.jobName" (dict "Release" .Release.Name "Name" "helmless" "Version" .Chart.Version "Values" .Values) -}}
{{- end }}

{{/*
Annotations for the webhooks
*/}}
{{- define "cloudzero-agent.webhooks.annotations" -}}
{{- if or .Values.insightsController.tls.useCertManager .Values.insightsController.webhooks.annotations }}
annotations:
{{- if .Values.insightsController.webhooks.annotations }}
{{ toYaml .Values.insightsController.webhooks.annotations | nindent 2}}
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
{{- include "cloudzero-agent.internal.resourceName" (dict "context" . "component" "webhook" "subcomponent" "certificate") }}
{{- end }}

{{/*
Name for the secret holding TLS certificates
*/}}
{{- define "cloudzero-agent.tlsSecretName" -}}
{{- .Values.insightsController.tls.secret.name | default (include "cloudzero-agent.internal.resourceName" (dict "context" . "component" "webhook" "subcomponent" "tls")) }}
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
Merge multiple dictionaries with string-aware overwrite logic.
Similar to mergeOverwrite, but treats empty strings as "unset" values (like null).
This is useful for merging resource configurations where empty strings indicate
that a value should not be set, allowing fallback to other sources.

Accepts a list of dictionaries to merge, with later dictionaries taking precedence
over earlier ones, but only for non-empty string values.

Example usage:
{{- include "cloudzero-agent.mergeStringOverwrite" (list
      .Values.components.aggregator.collector.resources
      .Values.aggregator.collector.resources
    ) }}

This will merge the two resource configurations, with the second one taking
precedence for any non-empty string values, but empty strings in the second
configuration will not overwrite values from the first configuration.
*/}}
{{- define "cloudzero-agent.mergeStringOverwrite" -}}
{{- $result := (dict) -}}
{{- range $dict := . -}}
  {{- if $dict -}}
    {{- range $key, $value := $dict -}}
      {{- if kindIs "map" $value -}}
        {{- /* Recursively merge nested dictionaries */ -}}
        {{- $existing := (get $result $key | default (dict)) -}}
        {{- $merged := (include "cloudzero-agent.mergeStringOverwrite" (list $existing $value) | fromYaml) -}}
        {{- $_ := set $result $key $merged -}}
      {{- else if and $value (ne $value "") -}}
        {{- /* Only set non-empty string values */ -}}
        {{- $_ := set $result $key $value -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- $result | toYaml -}}
{{- end -}}

{{- define "cloudzero-agent.maybeGenerateSection" -}}
{{- if and .value (not (empty .value)) -}}
{{- .name }}:
  {{- toYaml .value | nindent 2 }}
{{- end -}}
{{- end -}}

{{/*
Generate container command with special handling:
- null/not set: Uses provided default command array
- empty array []: No command output (uses image's default entrypoint)
- non-empty array: Uses the specified command

Usage: {{ include "cloudzero-agent.generateContainerCommand" (dict "command" .Values.components.agent.clusteredNode.command "default" (list "/app/cloudzero-alloy")) | nindent 10 }}
*/}}
{{- define "cloudzero-agent.generateContainerCommand" -}}
{{- $isEmptyArray := and (kindIs "slice" .command) (empty .command) -}}
{{- if not $isEmptyArray -}}
command:
  {{- if kindIs "invalid" .command }}
  {{- toYaml .default | nindent 2 }}
  {{- else }}
  {{- toYaml .command | nindent 2 }}
  {{- end }}
{{- end -}}
{{- end -}}

{{/*
Generate image configuration with defaults.
*/}}
{{- define "cloudzero-agent.generateImage" -}}
{{- $digest      := (.image.digest      | default .defaults.digest) -}}
{{- $tag         := (.image.tag         | default .defaults.tag) -}}
{{- $repository  := (.image.repository  | default .defaults.repository) -}}
{{- $pullPolicy  := (.image.pullPolicy  | default .defaults.pullPolicy) -}}
{{- if .compat -}}
{{- $digest      = (.compat.digest      | default .image.digest      | default .defaults.digest) -}}
{{- $tag         = (.compat.tag         | default .image.tag         | default .defaults.tag) -}}
{{- $repository  = (.compat.repository  | default .image.repository  | default .defaults.repository) -}}
{{- $pullPolicy  = (.compat.pullPolicy  | default .image.pullPolicy  | default .defaults.pullPolicy) -}}
{{- end -}}
{{- if $digest -}}
image: "{{ $repository }}@{{ $digest }}"
{{- else if $tag -}}
image: "{{ $repository }}:{{ $tag }}"
{{- end }}
{{ if $pullPolicy -}}
imagePullPolicy: "{{ $pullPolicy }}"
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
Generate labels with merge support, name, and optional component.

Parameters (dict):
- root: Chart context (.) (required)
- name: Application name for app.kubernetes.io/name label (required)
- component: Optional architectural component for app.kubernetes.io/component label (optional)
- labels: List of label dicts to merge in order, from lowest to highest priority (required)

Returns: Formatted "labels:\n  key: value" output

Merge behavior:
- Starts with base labels (app.kubernetes.io/instance, app.kubernetes.io/part-of, etc.)
- Merges each dict in the labels list in order (later values override earlier)
- Adds app.kubernetes.io/name LAST (highest priority, cannot be overridden)
- Adds app.kubernetes.io/component LAST if provided (highest priority, cannot be overridden)

Following Kubernetes recommended labels:
- app.kubernetes.io/name: The application (e.g., "server", "aggregator", "webhook-server")
- app.kubernetes.io/component: Optional architectural role (e.g., "gatherer", "processor")
- app.kubernetes.io/part-of: Set to "cloudzero-agent" in baseLabels

The caller controls the priority order by arranging the labels list. Common pattern:
  (list defaults commonLabels componentLabels)

Example:
  {{- include "cloudzero-agent.generateLabels" (dict
      "root" .
      "name" "server"
      "labels" (list
        .Values.defaults.labels
        .Values.commonMetaLabels
        .Values.components.agent.labels
      )
    ) | nindent 8 }}
*/}}
{{- define "cloudzero-agent.generateLabels" -}}
{{- $root := .root | required "root context required" -}}
{{- $name := .name | required "name parameter required" -}}
{{- $component := .component | default "" -}}
{{- $labelsList := .labels | required "labels list required" -}}

{{/* Start with base labels */}}
{{- $merged := include "cloudzero-agent.baseLabels" $root | fromYaml -}}

{{/* Merge each label dict in order (lowest to highest priority) */}}
{{- range $labelDict := $labelsList -}}
  {{- if $labelDict -}}
    {{- $merged = mergeOverwrite $merged $labelDict -}}
  {{- end -}}
{{- end -}}

{{/* Add name label LAST so it has highest priority and cannot be overridden */}}
{{- $merged = mergeOverwrite $merged (dict "app.kubernetes.io/name" $name) -}}

{{/* Add component label if provided */}}
{{- if $component -}}
  {{- $merged = mergeOverwrite $merged (dict "app.kubernetes.io/component" $component) -}}
{{- end -}}

{{/* Output formatted labels */}}
{{- if len $merged -}}
labels:
{{- $merged | toYaml | nindent 2 -}}
{{- end -}}
{{- end -}}

{{/*
Generate annotations with merge support.

Parameters (dict):
- root: Chart context (.) (required)
- annotations: List of annotation dicts to merge in order, from lowest to highest priority (required)

Returns: Formatted "annotations:\n  key: value" output

Merge behavior:
- Merges each dict in the annotations list in order (later values override earlier)
- The caller controls the priority order by arranging the annotations list

Example:
  {{- include "cloudzero-agent.generateAnnotations" (dict
      "root" .
      "annotations" (list
        .Values.defaults.annotations
        .Values.components.agent.annotations
        .Values.server.podAnnotations
        (dict "checksum/config" (include "cloudzero-agent.configChecksum" .))
      )
    ) | nindent 8 }}
*/}}
{{- define "cloudzero-agent.generateAnnotations" -}}
{{- $root := .root | required "root context required" -}}
{{- $annotationsList := .annotations | required "annotations list required" -}}

{{- $merged := dict -}}

{{/* Merge each annotation dict in order (lowest to highest priority) */}}
{{- range $annotDict := $annotationsList -}}
  {{- if $annotDict -}}
    {{- $merged = mergeOverwrite $merged $annotDict -}}
  {{- end -}}
{{- end -}}

{{/* Output formatted annotations */}}
{{- if len $merged -}}
annotations:
{{- $merged | toYaml | nindent 2 -}}
{{- end -}}
{{- end -}}

{{/*
Generate affinity sections
*/}}
{{- define "cloudzero-agent.generateAffinity" -}}
{{ $affinity := .default }}
{{- if .affinity -}}
{{ $affinity = merge (deepCopy .affinity) .default }}
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
{{- include "cloudzero-agent.maybeGenerateSection" (dict "name" "nodeSelector" "value" $nodeSelector) -}}
{{- end -}}

{{/*
Generate a pod disruption budget
*/}}
{{- define "cloudzero-agent.generatePodDisruptionBudget" -}}
{{- $replicas := int (.replicas | default .component.replicas | default 99999) -}}
{{- $defaults := .root.Values.defaults.podDisruptionBudget | default (dict) -}}
{{- $component := .component.podDisruptionBudget | default (dict) -}}

{{/* Determine if PDB is enabled - check component first, then defaults */}}
{{- $enabled := $defaults.enabled -}}
{{- if hasKey $component "enabled" -}}
  {{- $enabled = $component.enabled -}}
{{- end -}}

{{/* If component has ANY PDB setting, use entire component PDB; otherwise use defaults */}}
{{- $pdb := $defaults -}}
{{- if or (hasKey $component "minAvailable") (hasKey $component "maxUnavailable") -}}
  {{- $pdb = $component -}}
{{- end -}}

{{/* Validate PDB configuration regardless of enabled state */}}
{{- if and $pdb.minAvailable $pdb.maxUnavailable }}
{{- fail (printf "Pod disruption budget for %s cannot have both minAvailable and maxUnavailable set." .name) -}}
{{- end }}

{{/* Only create PDB if enabled and has some configuration */}}
{{- if and $enabled (or $pdb.minAvailable $pdb.maxUnavailable .root.Values.defaults.podDisruptionBudget) }}
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: {{ .name }}
  namespace: {{ .root.Release.Namespace }}
  {{- include "cloudzero-agent.generateLabels" (dict
      "root" .root
      "name" .componentName
      "labels" (list
        .root.Values.defaults.labels
        .root.Values.commonMetaLabels
      )
    ) | nindent 2 }}
  {{- include "cloudzero-agent.generateAnnotations" (dict
      "root" .root
      "annotations" (list
        .root.Values.defaults.annotations
      )
    ) | nindent 2 }}
spec:
  {{- if $pdb.minAvailable }}
  {{- if lt $replicas (int $pdb.minAvailable) -}}
  {{- fail (printf "Insufficient replicas in %s (%d) for pod disruption budget minAvailable (%v)" .name $replicas $pdb.minAvailable) -}}
  {{- end }}
  minAvailable: {{ $pdb.minAvailable }}
  {{- end }}
  {{- if $pdb.maxUnavailable }}
  maxUnavailable: {{ $pdb.maxUnavailable }}
  {{- end }}
  selector:
    matchLabels:
      {{- .matchLabels | nindent 6 }}
{{- end }}
{{- end -}}

{{/*
Generate resources block
Accepts a resources configuration object directly
Example usage:
{{- include "cloudzero-agent.generateResources" .Values.server.resources | nindent 12 }}
*/}}
{{- define "cloudzero-agent.generateResources" -}}
{{- if . -}}
  {{- $resources := . -}}
  {{- $cleanResources := dict -}}
  {{- if $resources.requests -}}
    {{- $cleanRequests := dict -}}
    {{- if and $resources.requests.cpu (ne $resources.requests.cpu "") -}}
      {{- $_ := set $cleanRequests "cpu" $resources.requests.cpu -}}
    {{- end -}}
    {{- if and $resources.requests.memory (ne $resources.requests.memory "") -}}
      {{- $_ := set $cleanRequests "memory" $resources.requests.memory -}}
    {{- end -}}
    {{- if $cleanRequests -}}
      {{- $_ := set $cleanResources "requests" $cleanRequests -}}
    {{- end -}}
  {{- end -}}
  {{- if $resources.limits -}}
    {{- $cleanLimits := dict -}}
    {{- if and $resources.limits.cpu (ne $resources.limits.cpu "") -}}
      {{- $_ := set $cleanLimits "cpu" $resources.limits.cpu -}}
    {{- end -}}
    {{- if and $resources.limits.memory (ne $resources.limits.memory "") -}}
      {{- $_ := set $cleanLimits "memory" $resources.limits.memory -}}
    {{- end -}}
    {{- if $cleanLimits -}}
      {{- $_ := set $cleanResources "limits" $cleanLimits -}}
    {{- end -}}
  {{- end -}}
  {{- if $cleanResources -}}
    {{- include "cloudzero-agent.maybeGenerateSection" (dict "name" "resources" "value" $cleanResources) -}}
  {{- end -}}
{{- end -}}
{{- end -}}

{{/*
Generate imagePullSecrets block
Accepts a dictionary with "root" (the top-level chart context) and "image" (the component's image configuration object)

Fallback chain:
1. Component-specific: .image.pullSecrets
2. Defaults: .root.Values.defaults.image.pullSecrets
3. Deprecated root-level: .root.Values.imagePullSecrets

Example usage:
{{- include "cloudzero-agent.generateImagePullSecrets" (dict
      "root" .
      "image" .Values.components.foo.image
    ) | nindent 6 }}
*/}}
{{- define "cloudzero-agent.generateImagePullSecrets" -}}
{{- include "cloudzero-agent.maybeGenerateSection" (dict
      "name" "imagePullSecrets"
      "value" (.image.pullSecrets | default .root.Values.defaults.image.pullSecrets | default .root.Values.imagePullSecrets)
    ) -}}
{{- end -}}

{{/*
Generate securityContext block from a merged configuration dictionary.
Accepts a dictionary containing the merged security context configuration.

For merging with defaults fallback (null values fall back to defaults):
{{- include "cloudzero-agent.generateSecurityContext" (mergeOverwrite
      (.Values.defaults.securityContext | default (dict))
      (.Values.components.miscellaneous.configLoader.securityContext | default (dict))
    ) | nindent 6 }}
*/}}
{{- define "cloudzero-agent.generateSecurityContext" -}}
{{- include "cloudzero-agent.maybeGenerateSection" (dict "name" "securityContext" "value" .) -}}
{{- end -}}

{{/*
Security Context Helper Functions

Kubernetes has two distinct security context types with different schemas:

1. Pod SecurityContext (spec.securityContext) - includes fsGroup,
   supplementalGroups, etc.
2. Container SecurityContext (spec.containers[].securityContext) - includes
   allowPrivilegeEscalation, capabilities, etc.

Rather than maintaining separate configuration properties for each type (which
would be heavy-handed and prevent container-specific overrides), these helper
functions filter a common security context object to include only properties
valid for each level.

This approach allows:

- Single configuration source (defaults.securityContext +
  components.*.securityContext)
- Proper schema validation (no fsGroup in containers, no capabilities in pods)
- Future flexibility for container-specific security contexts
- Clean separation of concerns

This is considered a temporary approach while we evaluate better patterns for
container-specific security context configuration. We may eventually implement a
system that allows per-container security context overrides.
*/}}

{{/*
Filter a map to only include specified properties.
Returns a JSON string containing only the properties that exist in the input map.
*/}}
{{- define "cloudzero-agent.filterProperties" -}}
  {{- $input := .input -}}
  {{- $properties := .properties -}}
  {{- $result := dict -}}
  {{- range $property := $properties -}}
    {{- if hasKey $input $property -}}
      {{- $_ := set $result $property (get $input $property) -}}
    {{- end -}}
  {{- end -}}
  {{- $result | toJson -}}
{{- end -}}

{{/*
Generate pod security context configuration.
Filters the input to only include properties valid for pod-level security contexts.

Pod-level security context properties (from k8s.json schema):
- appArmorProfile, fsGroup, fsGroupChangePolicy, runAsGroup, runAsNonRoot, runAsUser
- seLinuxChangePolicy, seLinuxOptions, seccompProfile, supplementalGroups
- supplementalGroupsPolicy, sysctls, windowsOptions
*/}}
{{- define "cloudzero-agent.generatePodSecurityContext" -}}
{{- if . -}}
{{- $podProperties := list
      "appArmorProfile"
      "fsGroup"
      "fsGroupChangePolicy"
      "runAsGroup"
      "runAsNonRoot"
      "runAsUser"
      "seLinuxChangePolicy"
      "seLinuxOptions"
      "seccompProfile"
      "supplementalGroups"
      "supplementalGroupsPolicy"
      "sysctls"
      "windowsOptions"
-}}
{{- include "cloudzero-agent.maybeGenerateSection" (dict
      "name" "securityContext"
      "value" ((include "cloudzero-agent.filterProperties" (dict "input" . "properties" $podProperties)) | fromJson)
    ) -}}
{{- end -}}
{{- end -}}

{{/*
Generate container security context configuration.
Filters the input to only include properties valid for container-level security contexts.

Container-level security context properties (from k8s.json schema):
- allowPrivilegeEscalation, appArmorProfile, capabilities, privileged, procMount
- readOnlyRootFilesystem, runAsGroup, runAsNonRoot, runAsUser, seLinuxOptions
- seccompProfile, windowsOptions
*/}}
{{- define "cloudzero-agent.generateContainerSecurityContext" -}}
{{- if . -}}
{{- $containerProperties := list
      "allowPrivilegeEscalation"
      "appArmorProfile"
      "capabilities"
      "privileged"
      "procMount"
      "readOnlyRootFilesystem"
      "runAsGroup"
      "runAsNonRoot"
      "runAsUser"
      "seLinuxOptions"
      "seccompProfile"
      "windowsOptions"
  -}}
{{- include "cloudzero-agent.maybeGenerateSection" (dict
      "name" "securityContext"
      "value" ((include "cloudzero-agent.filterProperties" (dict "input" . "properties" $containerProperties)) | fromJson)
    ) -}}
{{- end -}}
{{- end -}}

{{/*
Alloy/Prometheus Implementation Detection Helpers

These helpers determine which metrics collector implementation to use based on
the configuration and provide utilities for selecting the appropriate behavior.
*/}}

{{/*
Derive the agent mode from legacy properties if explicitly set, otherwise fall back to components.agent.mode.

The mode is derived as follows:
1. If legacy properties are explicitly set (non-null), derive from them:
   - defaults.federation.enabled=true -> "federated"
   - server.agentMode=true -> "agent"
   - server.agentMode=false -> "server"
2. Otherwise, use components.agent.mode (which has a default value)
   - components.agent.mode can be "federated", "agent", "server", or "clustered"
   - Note: The only way to use Alloy is by setting components.agent.mode to "clustered"

Returns one of: "federated", "agent", "server", "clustered"

Usage in templates: {{ eq (include "cloudzero-agent.Values.components.agent.mode" .) "federated" }}
*/}}
{{- define "cloudzero-agent.Values.components.agent.mode" -}}
  {{- /* If components.agent.mode is set (not null), it takes precedence over everything. */ -}}
  {{- if .Values.components.agent.mode -}}
    {{- .Values.components.agent.mode -}}
  {{- else -}}
    {{- /* Automatic mode: use legacy properties with default to agent */ -}}
    {{- if and .Values.defaults .Values.defaults.federation (eq .Values.defaults.federation.enabled true) -}}
      federated
    {{- else if and .Values.server (ne .Values.server.agentMode nil) (eq .Values.server.agentMode false) -}}
      server
    {{- else -}}
      {{- /* Default: Prometheus agent mode */ -}}
      agent
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Get the metrics collector image configuration

Returns the appropriate image configuration based on which collector is active:
- For Alloy: Uses components.agent.clusteredNode.image
- For Prometheus: Uses components.prometheus.image

Usage: {{ include "cloudzero-agent.agentCollectorImage" . }}
Returns: Image object with repository, tag, registry, pullPolicy
*/}}
{{- define "cloudzero-agent.agentCollectorImage" -}}
{{- if eq (include "cloudzero-agent.Values.components.agent.mode" .) "clustered" -}}
  {{- toYaml .Values.components.agent.clusteredNode.image -}}
{{- else -}}
  {{- toYaml .Values.components.prometheus.image -}}
{{- end -}}
{{- end -}}

{{/*
Get autoscaling configuration with fallback to defaults

Returns the autoscaling configuration, falling back to defaults.autoscaling
when the component-specific autoscaling is null or when individual properties
are not set.

Usage: {{ include "cloudzero-agent.getAutoscaling" (dict "component" .Values.components.agent.autoscaling "defaults" .Values.defaults.autoscaling) }}
Returns: Autoscaling configuration object
*/}}
{{- define "cloudzero-agent.getAutoscaling" -}}
{{- if .component -}}
  {{- $result := dict
        "enabled" (hasKey .component "enabled" | ternary .component.enabled .defaults.enabled)
        "minReplicas" (.component.minReplicas | default .defaults.minReplicas)
        "maxReplicas" (.component.maxReplicas | default .defaults.maxReplicas)
        "targetCPUUtilizationPercentage" (.component.targetCPUUtilizationPercentage | default .defaults.targetCPUUtilizationPercentage)
        "targetMemoryUtilizationPercentage" (.component.targetMemoryUtilizationPercentage | default .defaults.targetMemoryUtilizationPercentage)
  -}}
  {{- toYaml $result -}}
{{- else -}}
  {{- toYaml .defaults -}}
{{- end -}}
{{- end -}}

{{/*
Get the metrics collector container name

Returns the appropriate container name for the metrics collector:
- "alloy" for Alloy
- "prometheus" for Prometheus

Usage: {{ include "cloudzero-agent.agentCollectorContainerName" . }}
*/}}
{{- define "cloudzero-agent.agentCollectorContainerName" -}}
{{- if eq (include "cloudzero-agent.Values.components.agent.mode" .) "clustered" -}}
alloy
{{- else -}}
prometheus
{{- end -}}
{{- end -}}

{{/*
Get the metrics collector configuration file name

Returns the appropriate configuration file name:
- "alloy-config.river" for Alloy
- "prometheus.yml" for Prometheus

Usage: {{ include "cloudzero-agent.agentCollectorConfigFileName" . }}
*/}}
{{- define "cloudzero-agent.agentCollectorConfigFileName" -}}
{{- if eq (include "cloudzero-agent.Values.components.agent.mode" .) "clustered" -}}
alloy-config.river
{{- else -}}
prometheus.yml
{{- end -}}
{{- end -}}

{{/*
Resolve the Prometheus image tag.

Resolves the tag from components.prometheus.image.tag, falling back to
Chart.AppVersion with "-distroless" appended to use the official distroless
image variant (no shell, minimal attack surface).

Note: the deprecated server.image.tag compat override is handled by
generateImage's compat layer at the call site, not here.

Usage: {{ include "cloudzero-agent.Values.components.prometheus.image.tag" . }}
Returns: string (e.g., "v3.10.0-distroless", "v3.7.3")
*/}}
{{- define "cloudzero-agent.Values.components.prometheus.image.tag" -}}
  {{- .Values.components.prometheus.image.tag | default (printf "%s-distroless" .Chart.AppVersion) -}}
{{- end -}}

{{/*
Get the appropriate Prometheus agent mode flag based on version and mode

Determines whether Prometheus should run in agent mode and which flag to use:
- Prometheus 2.x uses --enable-feature=agent
- Prometheus 3.x uses --agent
- Returns empty string if not in agent/federated mode

The cloudzero-agent.Values.components.agent.mode helper already handles all the
complex mode derivation logic, so we just check if it returns "agent" or "federated"
and then determine the appropriate version-specific flag.

Uses the same tag fallback chain as image generation via
cloudzero-agent.Values.components.prometheus.image.tag

Usage: {{ include "cloudzero-agent.prometheusAgentFlag" . }}
Returns: string (either "--agent", "--enable-feature=agent", or empty string)
*/}}
{{- define "cloudzero-agent.prometheusAgentFlag" -}}
  {{- $mode := include "cloudzero-agent.Values.components.agent.mode" . -}}
  {{- if or (eq $mode "agent") (eq $mode "federated") -}}
    {{- $tag := include "cloudzero-agent.Values.components.prometheus.image.tag" . -}}
    {{- if hasPrefix "v2." $tag -}}
      --enable-feature=agent
    {{- else -}}
      --agent
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Istio Integration Detection Helper

Determines whether Istio integration should be enabled based on configuration
and cluster capabilities. Supports three modes:

- null (default): Auto-detect Istio via CRD presence in the cluster
- true: Force Istio integration enabled
- false: Force Istio integration disabled

When Istio is detected/enabled:
- If cluster ID is set: DestinationRule and VirtualService are created for cluster-local service isolation
- Webhook/backfill pods get port exclusion annotations (when not using cert-manager)

When Istio is NOT detected/disabled:
- No DestinationRule/VirtualService are created
- No Istio-specific annotations are added

Usage: {{ if include "cloudzero-agent.Values.integrations.istio.enabled" . }}...{{ end }}
Returns: "true" (truthy) when enabled, empty string (falsy) when disabled
*/}}
{{- define "cloudzero-agent.Values.integrations.istio.enabled" -}}
{{- $istioSetting := .Values.integrations.istio.enabled -}}
{{- if kindIs "invalid" $istioSetting -}}
  {{- /* null/not set = auto-detect via CRD presence */ -}}
  {{- if or (.Capabilities.APIVersions.Has "networking.istio.io/v1") (.Capabilities.APIVersions.Has "networking.istio.io/v1beta1") -}}
    {{- true -}}
  {{- end -}}
{{- else if $istioSetting -}}
  {{- /* true = force enabled */ -}}
  {{- true -}}
{{- end -}}
{{- /* false = force disabled, returns empty string */ -}}
{{- end -}}

{{/*
cAdvisor Integration Enabled Helper

Returns "true" if enabled, empty string if disabled (for use in conditionals).
- prometheusConfig.scrapeJobs.cadvisor.enabled takes priority if explicitly set
- integrations.cAdvisor.enabled is the fallback when null
*/}}
{{- define "cloudzero-agent.Values.integrations.cAdvisor.enabled" -}}
{{- $enabled := .Values.integrations.cAdvisor.enabled -}}
{{- if not (kindIs "invalid" .Values.prometheusConfig.scrapeJobs.cadvisor.enabled) -}}
{{- $enabled = .Values.prometheusConfig.scrapeJobs.cadvisor.enabled -}}
{{- end -}}
{{- if $enabled }}true{{- end -}}
{{- end -}}

{{/*
Istio Cluster ID Helper

Returns the Istio cluster ID to use for multicluster mesh configurations.
Falls back from integrations.istio.clusterID to clusterName.

This value is OPTIONAL. When set, DestinationRule and VirtualService resources
are created to ensure aggregator traffic stays within the local cluster.

If not explicitly set, falls back to clusterName. This allows automatic traffic
fencing in sidecar mode where we can validate the effective value at runtime.

The validator includes a runtime check that detects cross-cluster load balancing
and validates the effective cluster ID matches Istio's configuration.

Usage: {{ include "cloudzero-agent.istio.clusterID" . }}
Returns: The Istio cluster ID string (explicit or fallback to clusterName)
*/}}
{{- define "cloudzero-agent.istio.clusterID" -}}
{{- .Values.integrations.istio.clusterID | default .Values.clusterName -}}
{{- end -}}

{{/*
Validator Stage Helper

Generates a complete diagnostic stage configuration from values.yaml.
Each check value is one of: required, optional, informative, disabled.

Check types:
  - "required": failures cause non-zero exit code
  - "optional": failures logged but don't affect exit code
  - "informative": information gathering only, always passes
  - "disabled": check is not run

Arguments (passed as dict):
  - stage: The stage name (e.g., "pre-start", "post-start", "config-load")
  - checksConfig: The full checks map (.Values.components.validator.checks)

Usage: {{ include "cloudzero-agent.validator.stageCheck" (dict "stage" "pre-start" "checksConfig" .Values.components.validator.checks) }}
Returns: YAML stage object with name and checks fields
*/}}
{{- define "cloudzero-agent.validator.stageCheck" -}}
{{- $stage := .stage -}}
{{- $stageChecks := index .checksConfig $stage | default dict -}}
{{- $checks := list -}}
{{- range $checkName, $checkType := $stageChecks -}}
  {{/* Skip disabled checks */}}
  {{- if ne $checkType "disabled" -}}
    {{/* Default null/empty type to "optional" */}}
    {{- $effectiveType := $checkType | default "optional" -}}
    {{- $checks = append $checks (dict "name" $checkName "type" $effectiveType) -}}
  {{- end -}}
{{- end -}}
{{/* Output with consistent field order: name first, then checks */}}
name: {{ $stage }}
checks: {{ $checks | toYaml | nindent 2 -}}
{{- end -}}
