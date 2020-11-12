# Secrets Manager

Secrets Manager is a controller watches changes of secret in namespace and sync to
other namespaces with label `ops.dev/secret-sync=enabled`.

Secret with type will be watched:

* kubernetes.io/dockerconfigjson
* kubernetes.io/tls
* bootstrap.kubernetes.io/token
* Opaque

# Installation

## In Cluster as Deployment

```shell script
$ kubectl apply -f deploy/all-in-one.yaml
```

## Out of Cluster

```shell script
# compile
$ make build
# start
$ ./secrets-manager watch -n kube-secretmanager --kube-config <your-kube-config-path>
```

# TODO

* Upgrade ingress when related tls changes
* Auto configure deployment's pull secrets with dockerconfigjson via webhook
* Auto configure ingress tls if matched tls found

