---
title: "NodeFeatureDiscovery"
layout: default
sort: 2
---

# The NodeFeatureDiscovery CR

The `NodeFeatureDiscovery` CustomResource defines operational variables
to define the behaviour of the Node Feature Discovery Operand,
an example of the CustomResource:

```yaml
apiVersion: nfd.kubernetes.io/v1
kind: NodeFeatureDiscovery
metadata:
  name: nfd-master-server
  namespace: node-feature-discovery-operator
spec:
  instance: "" # instance is empty by default
  #labelWhiteList: ""
  #extraLabelNs:
  #  - "example.com"
  #resourceLabels:
  #  - "example.com/resource"
  operand:
    namespace: node-feature-discovery-operator
    image: gcr.io/k8s-staging-nfd/node-feature-discovery:master
    imagePullPolicy: Always
    servicePort: 12000
  workerConfig:
    configData: |
      #core:
      #  labelWhiteList:
      #  noPublish: false
      #  sleepInterval: 60s
      #  featureSources: [all]
      #  labelSources: [all]
      #  klog:
      #    addDirHeader: false
      #    alsologtostderr: false
      #    logBacktraceAt:
      #    logtostderr: true
      #    skipHeaders: false
      #    stderrthreshold: 2
      #    v: 0
      #    vmodule:
      ##   NOTE: the following options are not dynamically run-time configurable
      ##         and require a nfd-worker restart to take effect after being changed
      #    logDir:
      #    logFile:
      #    logFileMaxSize: 1800
      #    skipLogHeaders: false
      #sources:
      #  cpu:
      #    cpuid:
      ##     NOTE: whitelist has priority over blacklist
      #      attributeBlacklist:
      #        - "BMI1"
      #        - "BMI2"
      #        - "CLMUL"
      #        - "CMOV"
      #        - "CX16"
      #        - "ERMS"
      #        - "F16C"
      #        - "HTT"
      #        - "LZCNT"
      #        - "MMX"
      #        - "MMXEXT"
      #        - "NX"
      #        - "POPCNT"
      #        - "RDRAND"
      #        - "RDSEED"
      #        - "RDTSCP"
      #        - "SGX"
      #        - "SSE"
      #        - "SSE2"
      #        - "SSE3"
      #        - "SSE4"
      #        - "SSE42"
      #        - "SSSE3"
      #      attributeWhitelist:
      #  kernel:
      #    kconfigFile: "/path/to/kconfig"
      #    configOpts:
      #      - "NO_HZ"
      #      - "X86"
      #      - "DMI"
      #  pci:
      #    deviceClassWhitelist:
      #      - "0200"
      #      - "03"
      #      - "12"
      #    deviceLabelFields:
      #      - "class"
      #      - "vendor"
      #      - "device"
      #      - "subsystem_vendor"
      #      - "subsystem_device"
      #  usb:
      #    deviceClassWhitelist:
      #      - "0e"
      #      - "ef"
      #      - "fe"
      #      - "ff"
      #    deviceLabelFields:
      #      - "class"
      #      - "vendor"
      #      - "device"
      #  custom:
      #    # The following feature demonstrates the capabilities of the matchFeatures
      #    - name: "my custom rule"
      #      labels:
      #        my-ng-feature: "true"
      #      # matchFeatures implements a logical AND over all matcher terms in the
      #      # list (i.e. all of the terms, or per-feature matchers, must match)
      #      matchFeatures:
      #        - feature: cpu.cpuid
      #          matchExpressions:
      #            AVX512F: {op: Exists}
      #        - feature: cpu.cstate
      #          matchExpressions:
      #            enabled: {op: IsTrue}
      #        - feature: cpu.pstate
      #          matchExpressions:
      #            no_turbo: {op: IsFalse}
      #            scaling_governor: {op: In, value: ["performance"]}
      #        - feature: cpu.rdt
      #          matchExpressions:
      #            RDTL3CA: {op: Exists}
      #        - feature: cpu.sst
      #          matchExpressions:
      #            bf.enabled: {op: IsTrue}
      #        - feature: cpu.topology
      #          matchExpressions:
      #            hardware_multithreading: {op: IsFalse}
      #
      #        - feature: kernel.config
      #          matchExpressions:
      #            X86: {op: Exists}
      #            LSM: {op: InRegexp, value: ["apparmor"]}
      #        - feature: kernel.loadedmodule
      #          matchExpressions:
      #            e1000e: {op: Exists}
      #        - feature: kernel.selinux
      #          matchExpressions:
      #            enabled: {op: IsFalse}
      #        - feature: kernel.version
      #          matchExpressions:
      #            major: {op: In, value: ["5"]}
      #            minor: {op: Gt, value: ["10"]}
      #
      #        - feature: storage.block
      #          matchExpressions:
      #            rotational: {op: In, value: ["0"]}
      #            dax: {op: In, value: ["0"]}
      #
      #        - feature: network.device
      #          matchExpressions:
      #            operstate: {op: In, value: ["up"]}
      #            speed: {op: Gt, value: ["100"]}
      #
      #        - feature: memory.numa
      #          matchExpressions:
      #            node_count: {op: Gt, value: ["2"]}
      #        - feature: memory.nv
      #          matchExpressions:
      #            devtype: {op: In, value: ["nd_dax"]}
      #            mode: {op: In, value: ["memory"]}
      #
      #        - feature: system.osrelease
      #          matchExpressions:
      #            ID: {op: In, value: ["fedora", "centos"]}
      #        - feature: system.name
      #          matchExpressions:
      #            nodename: {op: InRegexp, value: ["^worker-X"]}
      #
      #        - feature: local.label
      #          matchExpressions:
      #            custom-feature-knob: {op: Gt, value: ["100"]}
```

For more information about how to setup the `WorkerConfig` stanza,
see
[worker config reference](https://kubernetes-sigs.github.io/node-feature-discovery/{{site.operand_version}}/advanced/worker-configuration-reference.html)
