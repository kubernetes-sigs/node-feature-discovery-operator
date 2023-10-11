FROM quay.io/operator-framework/helm-operator:v1.32.0

LABEL io.k8s.display-name="node-feature-discovery-operator"

ENV HOME=/opt/helm
COPY watches.yaml ${HOME}/watches.yaml
COPY nfd  ${HOME}/helm-charts
WORKDIR ${HOME}

# Run as unprivileged user
USER 65534:65534
