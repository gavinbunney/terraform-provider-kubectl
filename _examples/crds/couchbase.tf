provider "k8sraw" {}

resource "k8sraw_yaml" "test" {
  depends_on = ["k8sraw_yaml.definecrd"]

    yaml_body = <<YAML
apiVersion: couchbase.com/v1
kind: CouchbaseCluster
metadata:
  name: name-here-cluster
spec:
  baseImage: name-here-image
  version: name-here-image-version
  authSecret: name-here-operator-secret-name
  exposeAdminConsole: true
  adminConsoleServices:
    - data
  cluster:
    dataServiceMemoryQuota: 256
    indexServiceMemoryQuota: 256
    searchServiceMemoryQuota: 256
    eventingServiceMemoryQuota: 256
    analyticsServiceMemoryQuota: 1024
    indexStorageSetting: memory_optimized
    autoFailoverTimeout: 120
    autoFailoverMaxCount: 3
    autoFailoverOnDataDiskIssues: true
    autoFailoverOnDataDiskIssuesTimePeriod: 120
    autoFailoverServerGroup: false
  buckets:
    - name: default
      type: couchbase
      memoryQuota: 128
      replicas: 1
      ioPriority: high
      evictionPolicy: fullEviction
      conflictResolution: seqno
      enableFlush: true
      enableIndexReplica: false
    - name: test
      type: couchbase
      memoryQuota: 128
      replicas: 1
      ioPriority: high
      evictionPolicy: fullEviction
      conflictResolution: seqno
      enableFlush: true
      enableIndexReplica: false
  servers:
    - size: 3
      name: all_services
      services:
        - data
        - index
        - query
        - search
        - eventing
        - analytics

    YAML
}


resource "k8sraw_yaml" "definecrd" {
    yaml_body = <<YAML

    YAML
}
