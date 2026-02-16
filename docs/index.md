---
layout: home

hero:
  name: gateway
  image:
    src: /logo.png
    alt: gateway
  text: Application Gateway for Kubernetes
  tagline: A Go library for registering and routing services using Kubernetes Custom Resource Definitions.
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/foomo/gateway

features:
  - title: Kubernetes Native
    details: Define service registrations as Kubernetes Custom Resources and manage them with standard kubectl workflows.
  - title: Event-Driven
    details: Watch for Gateway CRD changes in real time with a simple event channel — add, update, and delete events streamed to your application.
  - title: Flexible Routing
    details: Configure path-based routing, base path stripping, CMS integration, internal access groups, and sitemap generation per service.
---
