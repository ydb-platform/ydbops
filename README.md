# ydbops

`ydbops` utility is used to perform various ad-hoc and maintenance operations on YDB clusters.

## On the up-to-date'ness of this readme

Soon an official documentation on [ydb.tech](https://ydb.tech) will be available. 

For now, please use the below info for reference only, it might be slightly outdated.

## Prerequisites:

1. Go 1.21

## How to build

Execute build.sh
Also Dockerfile can be used as a part of other multi-stage dockerfiles.

## How to run tests:

Ginkgo testing library is used. Do:

```
ginkgo test -vvv ./tests
```

## Current limitations:

1. [NON-CRITICAL FEATURE] Yandex IAM authorization with SA account key file is currently unsupported. However, you can always issue the token yourself and put it inside the `YDB_TOKEN` variable: `export YDB_TOKEN=$(ycp --profile <profile> iam create-token)`

## How to use:

Please browse the `ydbops --help` first. Then read along for examples (substitute your own values, of course).

#### Restart baremetal storage hosts

```
ydbops restart --storage \
  --endpoint grpc://<cluster-fqdn> \
  --ssh-args=pssh,-A,-J,<bastion-fqdn>,--ycp-profile,prod,--no-yubikey \
  --verbose --hosts=<node1-fqdn>,<node2-fqdn>,<node3-fqdn>
```

#### Restarting hosts without specifying filters will restart all of them

```
ydbops restart --storage \
  --endpoint grpc://<cluster-fqdn> \
  --ssh-args=pssh,-A,-J,<bastion-fqdn>,--ycp-profile,prod,--no-yubikey \
  --verbose
```

##### Run hello-world on remote hosts

```
ydbops run \
  --endpoint grpc://<cluster-fqdn> \
  --availability-mode strong --verbose --hosts=7,8 \
  --payload ./tests/payloads/payload-echo-helloworld.sh
```

##### Restart hosts using a custom payload

```
ydbops run \
  --endpoint grpc://<cluster-fqdn> \
  --availability-mode strong --verbose --hosts=5,6 \
  --payload ./tests/payloads/payload-restart-ydbd.sh
```

##### Restart storage in k8s

An example of authenticating with static credentials:

```
export YDB_PASSWORD=password_123
ydbops restart --storage \
  --endpoint grpc://<cluster-fqdn> \
  --availability-mode strong --verbose --hosts=7,8 \
  --user jorres --kubeconfig ~/.kube/config
```
