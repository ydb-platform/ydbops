# ydbops

`ydbops` utility is used to perform various ad-hoc and maintenance operations on YDB clusters.

## Prerequisites:

1. Go 1.21

## How to run tests:

Ginkgo testing library is used. Do:

```
ginkgo test -vvv ./tests
```

## Current limitations:

1. [CRITICAL FEATURE] Drain API is not in public YDB Maintenance GRPC api yet. Therefore, `ydbops` currently relies on builtin drain when restarting nodes (which has a certain timeout, and a node with a lot of tablets will probably not shutdown healthily). Will be implemented as soon as 
1. [NON-CRITICAL FEATURE] Yandex IAM authorization with SA account key file is currently unsupported. However, you can always issue the token yourself and put it inside the `YDB_TOKEN` variable: `export YDB_TOKEN=$(ycp --profile <profile> iam create-token)`

## How to use:

Please browse the `ydbops --help` first. Then read along for examples (substitute your own values, of course).

#### Restart baremetal storage hosts

```
ydbops restart --storage \
  --endpoint grpc://<cluster-fqdn> \
  --ssh-args=pssh,-A,-J,<bastion-fqdn>,--ycp-profile,prod,--no-yubikey \
  --verbose --hosts <node1-fqdn>,<node2-fqdn>,<node3-fqdn>
```

##### Run hello-world on remote hosts

```
ydbops run \
  --endpoint grpc://<cluster-fqdn> \
  --availability-mode strong --verbose --hosts 7,8 \
  --payload ./tests/payloads/payload-echo-helloworld.sh
```

##### Restart hosts using a custom payload

```
ydbops run \
  --endpoint grpc://<cluster-fqdn> \
  --availability-mode strong --verbose --hosts 5,6 \
  --payload ./tests/payloads/payload-restart-ydbd.sh
```

##### Restart storage in k8s

An example of authenticating with static credentials:

```
export YDB_PASSWORD=password_123
ydbops restart --storage \
  --endpoint grpc://<cluster-fqdn> \
  --availability-mode strong --verbose --hosts 7,8 \
  --user jorres --kubeconfig ~/.kube/config
```
