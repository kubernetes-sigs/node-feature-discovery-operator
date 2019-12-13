FROM registry.access.redhat.com/ubi8/go-toolset AS builder
WORKDIR /go/src/github.com/kubernetes-sigs/node-feature-discovery-operator
COPY . .
RUN make build

FROM registry.access.redhat.com/ubi8/ubi
COPY --from=builder /go/src/github.com/kubernetes-sigs/node-feature-discovery-operator/node-feature-discovery-operator /usr/bin/

RUN mkdir -p /etc/kubernetes/node-feature-discovery/assets
COPY assets/ /etc/kubernetes/node-feature-discovery/assets

#ADD controller-manifests /manifests


RUN useradd node-feature-discovery-operator
USER node-feature-discovery-operator
ENTRYPOINT ["/usr/bin/node-feature-discovery-operator"]
