---
name: New Release
about: Propose a new release
title: Release v0.x.0
assignees: ArangoGutierrez, marquiz, zvonkok

---

## Release Checklist
<!--
Please do not remove items from the checklist
-->
- [ ] All [OWNERS](https://github.com/kubernetes-sigs/node-feature-discovery-operator/blob/master/OWNERS) must LGTM the release proposal
- [ ] Verify that the changelog in this issue is up-to-date
- [ ] For major releases (v0.$MAJ.0), an OWNER creates a release branch with
      `git branch release-0.$MAJ master`
- [ ] An OWNER creates a vanilla release branch from master and pushes it with
      `git push release-0.$MAJ`
- [ ] An OWNER creates an annotated and signed tag with
     `git tag -a -s $VERSION`
      and inserts the changelog into the tag description.
- [ ] An OWNER pushes the tag with
      `git push $VERSION`
      This will trigger prow to build and publish a staging container image
      `gcr.io/k8s-staging-nfd/node-feature-discovery-operator:$VERSION`
- [ ] Submit a PR against [k8s.io](https://github.com/kubernetes/k8s.io), updating `k8s.gcr.io/images/k8s-staging-nfd/images.yaml` to promote the container image to production
- [ ] Wait for the PR to be merged and verify that the image (`k8s.gcr.io/nfd/node-feature-discovery-operator:$VERSION`) is available.
- [ ] Write the change log into the [Github release info](https://github.com/kubernetes-sigs/node-feature-discovery-operator/releases).
- [ ] Add a link to the tagged release in this issue.
- [ ] Create a new bundle for the $VERSION release at https://github.com/k8s-operatorhub/community-operators 
- [ ] Send an announcement email to `kubernetes-dev@googlegroups.com` with the subject `[ANNOUNCE] node-feature-discovery-operator $VERSION is released`
- [ ] Add a link to the release announcement in this issue
- [ ] Close this issue


## Changelog
<!--
Describe changes since the last release here.
-->
