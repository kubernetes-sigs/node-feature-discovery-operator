# Release Process

The process to release a new version of node-feature-discovery-operator is as follows:

- [ ] File [a new issue](https://github.com/kubernetes-sigs/node-feature-discovery-operator/issues/new)
  to propose a new release. Copy this checklist into the issue description
- [ ] Add a changelog section in the issue description, capturing changes since the
  previous release
- [ ] An OWNER runs `git tag -s $VERSION` and inserts the changelog into the
  tag description
- [ ] An OWNER pushes the tag with `git push $VERSION` - this will trigger prow
  to build and publish a staging container image
  `gcr.io/k8s-staging-nfd/node-feature-discovery-operator:$VERSION`
- [ ] Do final release verification on the staging image
- [ ] Submit a PR against [k8s.io](https://github.com/kubernetes/k8s.io),
  updating `k8s.gcr.io/images/k8s-staging-nfd/images.yaml`, in order to promote
  the container image to production
- [ ] Wait for the PR to be merged and verify that the image
  (`k8s.gcr.io/nfd/node-feature-discovery-operator:$VERSION`) is available
- [ ] Write the change log into the
  [Github release info](https://github.com/kubernetes-sigs/node-feature-discovery-operator/releases).
- [ ] Add a link to the tagged release in this issue
- [ ] Send an announcement email to `kubernetes-dev@googlegroups.com` with the
  subject `[ANNOUNCE] node-feature-discovery-operator $VERSION is released`
- [ ] Add a link to the release announcement email in this issue
- [ ] Close this issue
