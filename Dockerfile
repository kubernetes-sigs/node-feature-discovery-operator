# Build the manager binary
FROM golang:1.16.6-buster as builder

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
FROM registry.access.redhat.com/ubi8/ubi
COPY --from=builder /workspace/node-feature-discovery-operator /

RUN mkdir -p /opt/nfd
COPY build/assets /opt/nfd

RUN useradd nfd-operator
USER nfd-operator

ENTRYPOINT ["/node-feature-discovery-operator"]
LABEL io.k8s.display-name="node-feature-discovery-operator"
