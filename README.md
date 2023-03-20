# Node Feature Discovery Operator

The Node Feature Discovery operator is a tool for Kubernetes administrators 
that makes it easy to detect and understand the hardware features and 
configurations of a cluster's nodes. With this operator, administrators can 
easily gather information about their nodes that can be used for scheduling, 
resource management, and more by controlling the life cycle of 
[NFD](https://github.com/kubernetes-sigs/node-feature-discovery).

## How it Works

The operator works by orchestrating all resources needed to run the 
Node-Feature-Discovery (NFD). NFD runs on each node in the cluster and detects 
the features and configurations of the node's hardware.

## Quick start


Get the source code

```bash
git clone -b v0.6.0 https://github.com/kubernetes-sigs/node-feature-discovery-operator
```

Deploy the operator

> By default it will deploy using the minimal tag image, is
> desired you can simply modify the IMAGE_TAG env var to point to the image
> tag to use.

```bash
IMAGE_TAG={{ site.container_image }}
make deploy
```

By default the operator will watch `NodeFeatureDiscovery` objects
only in the namespace where the operator is deployed in. This is
specified by the `WATCH_NAMESPACE` env variable in the operator
deployment manifest. If unset the operator will watch ALL
namespaces.

Create a NodeFeatureDiscovery instance

```bash
kubectl apply -f config/samples/nfd.kubernetes.io_v1_nodefeaturediscovery.yaml
```

## Documentation

For more detailed information on how to use the Node Feature Discovery operator,
please check out our 
[documentation](https://kubernetes-sigs.github.io/node-feature-discovery-operator/master)

## Contributing

The Node Feature Discovery operator welcomes contributions, and interested 
parties are encouraged to take a look at the 
[contributing guidelines](CONTRIBUTING.md) and 
[open issues](https://github.com/kubernetes-sigs/node-feature-discovery-operator/issues). 
We're excited to have you join our community of contributors.

## Support

If there are any issues or questions about the Node Feature Discovery operator,
they can be addressed by opening an issue on the 
[GitHub repository](https://github.com/kubernetes-sigs/node-feature-discovery-operator/issues/new/choose) 
or reaching out on the 
[Slack channel](https://kubernetes.slack.com/messages/node-feature-discovery).
