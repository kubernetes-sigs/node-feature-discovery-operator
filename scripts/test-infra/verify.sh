#!/bin/bash -e

curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
helm lint deploy/helm/nfd-operator --strict