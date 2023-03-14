---
title: "Introduction"
layout: default
sort: 1
---

# Node Feature Discovery Operator

Welcome to Node Feature Discovery Operator – an Operator
Framework implementation around the Node Feature Discovery project to enable
detecting hardware features and system configuration!

Continue to:

- **[Deployment](/deployment)** for instructions on how to
  deploy NFD-Operator to a cluster.

- **[Advanced](/advanced)** for more advanced topics and
  reference.

# Introduction

The Node Feature Discovery Operator manages the detection
of hardware features and configuration in a Kubernetes
cluster by labeling the nodes with hardware-specific information.
The Node Feature Discovery (NFD) will label the host with
node-specific attributes,
like PCI cards, kernel, or OS version, and many more.

The NFD Operator is based on the [Operator Framework](https://operatorframework.io/)
an open source toolkit to manage Kubernetes native applications, called
Operators, in an effective, automated, and scalable way.
