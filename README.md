
[![codecov](https://codecov.io/gh/toughnoah/elastalert-operator/branch/master/graph/badge.svg?token=5B1DBTNIDN)](https://codecov.io/gh/toughnoah/elastalert-operator) [![CI Workflow](https://github.com/toughnoah/elastalert-operator/actions/workflows/test-coverage.yaml/badge.svg)](https://github.com/toughnoah/elastalert-operator/actions/workflows/test-coverage.yaml)
# Elastalert Operator for Kubernetes

The Elastalert Operator is an implementation of a [Kubernetes Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

## Getting started

Firstly, learn [how to use elastalert](https://elastalert.readthedocs.io/en/latest/). Exactly how to setup a `config.yaml` and `rule`, the default command to start elastalert container is  `elastalert --config /etc/elastalert/config.yaml --verbose`

To install the operator, please refer yaml in`deploy` directory.

Once the `elastalert-operator` deployment in the namespace `alert` is ready, create a Elastalert instance, like:

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

This will create an elastalert instance named `elastalert`, and the operator will create and deployment named the same, like:

```console
# kubectl get -n alert deployment
NAME              READY   UP-TO-DATE   AVAILABLE   AGE
elastalert   1/1     1            1           10m
```
Next you can check you elastalert `pod`.
```console
# kubectl get -n alert pod
NAME                               READY   STATUS    RESTARTS   AGE
elastalert-76c95597f8-zmbzz   1/1     Running   0          116m
```



