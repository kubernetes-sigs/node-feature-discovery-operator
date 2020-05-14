# Build the operator
FROM registry.access.redhat.com/ubi8/go-toolset AS builder
WORKDIR /go/src/github.com/kubernetes-sigs/node-feature-discovery-operator
COPY . .

# do the actual build
RUN make build

# Create production image for running the operator
FROM registry.access.redhat.com/ubi8/ubi
COPY --from=builder /go/src/github.com/kubernetes-sigs/node-feature-discovery-operator/node-feature-discovery-operator /usr/bin/

RUN mkdir -p /opt/nfd
COPY assets /opt/nfd

RUN useradd node-feature-discovery-operator
USER node-feature-discovery-operator
ENTRYPOINT ["/usr/bin/node-feature-discovery-operator"]
LABEL io.k8s.display-name="node-feature-discovery-operator" 
