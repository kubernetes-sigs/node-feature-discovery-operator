---
title: "Image variants"
layout: default
sort: 3
---

# Image variants

{: .no_toc}

## Table of contents

{: .no_toc .text-delta}

1. TOC
{:toc}

---

# Image variants

Node-Feautre-Discovery-Operator currently offers two variants
of the container image. The "full" variant is currently
deployed by default.

## Default

This is a minimal image based on:
[gcr.io/distroless/base](https://github.com/GoogleContainerTools/distroless/blob/master/base/README.md)

The container image tag has suffix `-minimal`
(e.g. `{{ site.container_image }}-minimal`)
and the image is deployed by default.

## Full

This image is based on
[debian:buster-slim](https://hub.docker.com/_/debian) and contains a full Linux
system for doing live debugging and diagnosis of the operator.
