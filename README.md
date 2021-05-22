
[![codecov](https://codecov.io/gh/toughnoah/elastalert-operator/branch/master/graph/badge.svg?token=5B1DBTNIDN)](https://codecov.io/gh/toughnoah/elastalert-operator) [![CI Workflow](https://github.com/toughnoah/elastalert-operator/actions/workflows/test-coverage.yaml/badge.svg)](https://github.com/toughnoah/elastalert-operator/actions/workflows/test-coverage.yaml)
# Elastalert Operator for Kubernetes

The Elastalert Operator is an implementation of a [Kubernetes Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

## Getting started

Firstly, learn [How to use elastalert](https://elastalert.readthedocs.io/en/latest/), exactly how to setup a `config.yaml` and `rule`.
The default command to start elastalert container is  `elastalert --config /etc/elastalert/config.yaml --verbose`.

To install the operator, please refer to those yamls in`deploy` directory.

Once the `elastalert-operator` deployment in the namespace `alert` is ready, create an Elastalert instance, like:

```
kubectl apply -n alert -f - <<EOF
apiVersion: es.noah.domain/v1alpha1
kind: Elastalert
metadata:
  name: elastalert
spec:
  rule:
  - name: error-messages
    type: any
    index: your-index-here
    filter:
    - query:
        query_string:
          query: "message: error"
    min_threshold: 1
  - name: error-status-code
    type: any
    index: your-index-here
    filter:
    - query:
        query_string:
          query: "http_status_code: 500"
    min_threshold: 1
  config:
    es_host: es.domain
    es_port: 9200
    es_username: username
    es_password: password
    writeback_index: your-elastalert-index
    run_every: 
      minutes: 5
    buffer_time: 
      minutes: 5
  overall:
    alert:
      - "post"
      http_post_url: "test.com"
      http_post_headers:
        Content-Type: |-
          "application/json"
      http_post_static_payload:
          ...
      
EOF
```

This will create an elastalert instance named `elastalert`, and the operator will create a deployment named the same, as:

```console
# kubectl get -n alert deployment
NAME              READY   UP-TO-DATE   AVAILABLE   AGE
elastalert        1/1     1            1           10m
```
Next you can check your elastalert `pod`.
```console
# kubectl get -n alert pod
NAME                               READY   STATUS    RESTARTS   AGE
elastalert-76c95597f8-zmbzz         1/1    Running      0       10m
```
And check your configmaps that `-config` is created from config. `-rule` is created from rule.
```console
# kubectl get configmap
NAME                     DATA   AGE
elastalert-config         1     10m
elastalert-rule           1     10m
```
Above mentioned configmaps will be mounted to `/etc/elastalert` and `/etc/elastalert/rule` as `config.yaml` and `rule` yaml named by its `["name"]`
```console
/etc/elastalert # ls
config.yaml  rules
```
```console
/etc/elastalert/rules # ls
error-message.yaml      error-status-code.yaml
```
## What's more
### Elasticsearch Cert 
```
kubectl apply -n alert -f - <<EOF
apiVersion: es.noah.domain/v1alpha1
kind: Elastalert
metadata:
  name: elastalert
spec:
  rule:
  - name: error-messages
    ...
  - name: error-status-code
    ...
  config:
    use_ssl: True
  overall:
    alert:
      ...
  cert: |-
        -----BEGIN CERTIFICATE-----
      MIIDYjCCAkqgAwIBAgIRAPUiB7MQDUsQlH5qOvwzaeQwDQYJKoZIhvcNAQELBQAw
      OzEZMBcGA1UECxMQc2ctZWxhc3RpY3NlYXJjaDEeMBwGA1UEAxMVc2ctZWxhc3Rp
      Y3NlYXJjaC1odHRwMB4XDTIwMTIxNjA2MTYyMloXDTIxMTIxNjA2MjYyMlowOzEZ
      MBcGA1UECxMQc2ctZWxhc3RpY3NlYXJjaDEeMBwGA1UEAxMVc2ctZWxhc3RpY3Nl
      YXJjaC1odHRwMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv6wgW0Al
      6txmJN0/ihVZkqxtmY9owIskB3CxOaaXva27LGqj0Rp5b/OXJIem2h9UKuYqTQCp
      xwToEVw7rmIuyGXRtC2qSKHDFDSFHSDJFHSDKFSDHKCTi6O6uqMcrgjWwATfT76i
      dzeceg4Ly8FINTd2MBi5IbB+UybT/V0T89CeRssjQHEjAX5qEJYW7iTG01nAvaVl
      qfmsMvsZVzU+T8K7ZWrfhkRI2y6ln3hfsE1rKeAzX788RwU8o3GA41Jk5md0yE3a
      y5h0odVoanM1GLH29nswl28t0UAvq/K7kg38V0Kzuocc09mStCXKTf+I74YG7LBu
      V8IHPfCliPSvCwSKDHFJSDHFJKSDNMXCWER72sfSDFETVNFQAheqoQwHQYDVR0lB
      KwYBBQUHAwEGCCsGAQUFBwMCMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFGVU
      OaiBC+86H7lAHu5xM95vSO+VMA0GCSqGSIb3DQEBCwUAA4IBAQCM8mvNRjDOj/kn
      7Fni7FVp6v7Oa7yXiK0knzRX9GoHkniA/a5rZN3Fau+i2y6g2vaUs4BtdsAAdyC7
      GHIImn2M91nJXxcCFD0sfSKDJHFJSKDHFsdfsfeuWwMZJpBHSIw4aNmfX3l4c
      6D4+FzlWBKg/JTCv/63HVJ+m+HMKDE8h12aUM1n2rTHiMtnuRkIBa9uoXydN+QCM
      trjLH8AHqXNEPpraKFPRsCVHHhlBlDfpTSkvFgQLAKCh+heXDLyG1e+NEeU+sw9X
      sVz/o0pQkfwUNQjKsPwxQCHxjjw0qZn02wwX8fHgvmCwNIOK/WgsUlNDpny8CLiC
      pxb0oEG7
      -----END CERTIFICATE-----
EOF
```
This action will create a new `secret` mounted as `/ssl/elasticCA.crt`

### Overall
`overall` is used to config global alert settings. If you defined `alert` in a rule, it will override `overall` settings.
```
kubectl apply -n alert -f - <<EOF
apiVersion: es.noah.domain/v1alpha1
kind: Elastalert
metadata:
  name: elastalert
spec:
  rule:
  - name: error-messages
    ...
  - name: error-status-code
    ...
  config:
    ...
  overall:
    alert:
      - "post"
      http_post_url: "test.com"
      http_post_headers:
        Content-Type: |-
          "application/json"
      http_post_static_payload:
          ...
  cert: |-
    ...
EOF
```
### Pod Template
Define customized podTemplate
```
kubectl apply -n alert -f - <<EOF
apiVersion: es.noah.domain/v1alpha1
kind: Elastalert
metadata:
  name: elastalert
spec:
  rule:
    ...
  config:
    ...
  overall:
    ...
  cert: |-
    ...
  podTemplate:
      spec:
        containers:
        - name: elastalert
          imagePullPolicy: Always
          resources:
            limits:
              cpu: "2000m"
              memory: "4Gi"
            requests:
              cpu: "500m"
              memory: "1Gi" 
EOF
```
You can override the default command here.

### Notice
You don't have to specify `rules_folder` in config section, because operator will auto patch `rules_folder: /etc/elastalert/rules/..data/` for your config.
The reason why have to be `..data/` is the workaround when the configmap is mounted as file(such as `/etc/elastalert/rules/test.yaml`) in a pod, it will create a soft-link to `/etc/elastalert/rules/..data/test.yaml`.
That is to say, you will receive duplicated rules name error that both files in `rules` and `..data` would be loaded if you specify merely `rules_folder: /etc/elastalert/rules`

## Contact Me
Any advice is welcome! Please email to toughnoah@163.com
