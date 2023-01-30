ARG BUILDER_IMAGE
ARG BASE_IMAGE_FULL
ARG BASE_IMAGE_MINIMAL
# Build the manager biinary
FROM ${BUILDER_IMAGE} as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.sum ./

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Build
COPY . .
RUN make build

# Create production image for running the operator
FROM ${BASE_IMAGE_FULL} as full
COPY --from=builder /workspace/node-feature-discovery-operator /
COPY --from=builder /workspace/build/assets /opt/nfd

# Run as unprivileged user
USER 65534:65534

ENTRYPOINT ["/node-feature-discovery-operator"]
LABEL io.k8s.display-name="node-feature-discovery-operator"

# Create a minimal image for running the operator
FROM ${BASE_IMAGE_MINIMAL} as minimal
COPY --from=builder /workspace/node-feature-discovery-operator /
COPY --from=builder /workspace/build/assets /opt/nfd

ENTRYPOINT ["/node-feature-discovery-operator"]
LABEL io.k8s.display-name="node-feature-discovery-operator"

