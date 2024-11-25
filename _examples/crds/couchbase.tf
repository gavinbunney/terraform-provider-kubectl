provider "kubectl" {
  apply_retry_count = 5
}

resource "kubectl_manifest" "test" {
  depends_on = ["kubectl_manifest.definecrd"]

    yaml_body = <<YAML
apiVersion: couchbase.com/v1
kind: CouchbaseCluster
metadata:
  name: name-here-cluster
spec:
  baseImage: name-here-image
  version: 6.0.0-1
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


resource "kubectl_manifest" "definecrd" {
    yaml_body = <<YAML
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: couchbaseclusters.couchbase.com
spec:
  conversion:
    strategy: None
  group: couchbase.com
  names:
    kind: CouchbaseCluster
    listKind: CouchbaseClusterList
    plural: couchbaseclusters
    shortNames:
    - couchbase
    - cbc
    singular: couchbasecluster
  scope: Namespaced
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              adminConsoleServices:
                items:
                  enum:
                  - data
                  - index
                  - query
                  - search
                  - eventing
                  - analytics
                  type: string
                type: array
              antiAffinity:
                type: boolean
              authSecret:
                minLength: 1
                type: string
              baseImage:
                type: string
              buckets:
                items:
                  properties:
                    conflictResolution:
                      enum:
                      - seqno
                      - lww
                      type: string
                    enableFlush:
                      type: boolean
                    enableIndexReplica:
                      type: boolean
                    evictionPolicy:
                      enum:
                      - valueOnly
                      - fullEviction
                      - noEviction
                      - nruEviction
                      type: string
                    ioPriority:
                      enum:
                      - high
                      - low
                      type: string
                    memoryQuota:
                      minimum: 100
                      type: integer
                    name:
                      pattern: ^[a-zA-Z0-9._\-%]*$
                      type: string
                    replicas:
                      maximum: 3
                      minimum: 0
                      type: integer
                    type:
                      enum:
                      - couchbase
                      - ephemeral
                      - memcached
                      type: string
                  required:
                  - name
                  - type
                  - memoryQuota
                  type: object
                type: array
              cluster:
                properties:
                  analyticsServiceMemoryQuota:
                    minimum: 1024
                    type: integer
                  autoFailoverMaxCount:
                    maximum: 3
                    minimum: 1
                    type: integer
                  autoFailoverOnDataDiskIssues:
                    type: boolean
                  autoFailoverOnDataDiskIssuesTimePeriod:
                    maximum: 3600
                    minimum: 5
                    type: integer
                  autoFailoverServerGroup:
                    type: boolean
                  autoFailoverTimeout:
                    maximum: 3600
                    minimum: 5
                    type: integer
                  clusterName:
                    type: string
                  dataServiceMemoryQuota:
                    minimum: 256
                    type: integer
                  eventingServiceMemoryQuota:
                    minimum: 256
                    type: integer
                  indexServiceMemoryQuota:
                    minimum: 256
                    type: integer
                  indexStorageSetting:
                    enum:
                    - plasma
                    - memory_optimized
                    type: string
                  searchServiceMemoryQuota:
                    minimum: 256
                    type: integer
                required:
                - dataServiceMemoryQuota
                - indexServiceMemoryQuota
                - searchServiceMemoryQuota
                - eventingServiceMemoryQuota
                - analyticsServiceMemoryQuota
                - indexStorageSetting
                - autoFailoverTimeout
                - autoFailoverMaxCount
                type: object
              disableBucketManagement:
                type: boolean
              exposeAdminConsole:
                type: boolean
              exposedFeatures:
                items:
                  enum:
                  - admin
                  - xdcr
                  - client
                  type: string
                type: array
              logRetentionCount:
                minimum: 0
                type: integer
              logRetentionTime:
                pattern: ^\d+(ns|us|ms|s|m|h)$
                type: string
              paused:
                type: boolean
              serverGroups:
                items:
                  type: string
                type: array
              servers:
                items:
                  properties:
                    name:
                      minLength: 1
                      pattern: ^[-_a-zA-Z0-9]+$
                      type: string
                    pod:
                      properties:
                        automountServiceAccountToken:
                          type: boolean
                        couchbaseEnv:
                          items:
                            properties:
                              name:
                                type: string
                              value:
                                type: string
                            type: object
                          type: array
                        labels:
                          type: object
                        nodeSelector:
                          type: object
                        resources:
                          properties:
                            limits:
                              properties:
                                cpu:
                                  type: string
                                memory:
                                  type: string
                                storage:
                                  type: string
                              type: object
                            requests:
                              properties:
                                cpu:
                                  type: string
                                memory:
                                  type: string
                                storage:
                                  type: string
                              type: object
                          type: object
                        tolerations:
                          items:
                            properties:
                              effect:
                                type: string
                              key:
                                type: string
                              operator:
                                type: string
                              tolerationSeconds:
                                type: integer
                              value:
                                type: string
                            required:
                            - key
                            - operator
                            - value
                            - effect
                            type: object
                          type: array
                        volumeMounts:
                          properties:
                            analytics:
                              items:
                                type: string
                              type: array
                            data:
                              type: string
                            default:
                              type: string
                            index:
                              type: string
                            logs:
                              type: string
                          type: object
                      type: object
                    serverGroups:
                      items:
                        type: string
                      type: array
                    services:
                      items:
                        enum:
                        - data
                        - index
                        - query
                        - search
                        - eventing
                        - analytics
                        type: string
                      minLength: 1
                      type: array
                    size:
                      minimum: 1
                      type: integer
                  required:
                  - size
                  - name
                  - services
                  type: object
                minLength: 1
                type: array
              softwareUpdateNotifications:
                type: boolean
              tls:
                properties:
                  static:
                    properties:
                      member:
                        properties:
                          serverSecret:
                            type: string
                        type: object
                      operatorSecret:
                        type: string
                    type: object
                type: object
              version:
                pattern: ^([\w\d]+-)?\d+\.\d+.\d+(-[\w\d]+)?$
                type: string
              volumeClaimTemplates:
                items:
                  properties:
                    metadata:
                      properties:
                        name:
                          type: string
                      required:
                      - name
                      type: object
                    spec:
                      properties:
                        resources:
                          properties:
                            limits:
                              properties:
                                storage:
                                  type: string
                              required:
                              - storage
                              type: object
                            requests:
                              properties:
                                storage:
                                  type: string
                              required:
                              - storage
                              type: object
                          type: object
                        storageClassName:
                          type: string
                      required:
                      - resources
                      - storageClassName
                      type: object
                  required:
                  - metadata
                  - spec
                  type: object
                type: array
            required:
            - baseImage
            - version
            - authSecret
            - cluster
            - servers
    YAML
}
