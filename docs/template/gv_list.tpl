{{- define "gvList" -}}
{{- $groupVersions := . -}}

---
title: Resource Definitions API Reference
slug: /reference/api7-ingress-controller/crd-reference
description: Explore detailed reference documentation for the custom resource definitions (CRDs) supported by the API7 Ingress Controller.
---

This document provides the API resource description for the API7 Ingress Controller.

## Packages
{{- range $groupVersions }}
- {{ markdownRenderGVLink . }}
{{- end }}

{{ range $groupVersions }}
{{ template "gvDetails" . }}
{{ end }}

{{- end -}}
