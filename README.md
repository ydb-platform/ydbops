# ydbops

`ydbops` utility is used to perform various ad-hoc and maintenance operations on YDB clusters.

For comprehensive documentation, refer to [ydb.tech](https://ydb.tech/docs/en/reference/ydbops/)

## Quick non-comprehensive cheatsheet:

Please browse the `ydbops --help` first. Then read along for examples (substitute your own values).

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

##### Restart tenant in k8s concurrently and nodes ordered by tenant


An example of concurrent restarts will spawn 3 goroutines for node restarts. it will make sure to not restart nodes from more than 2 tenants at the same time:

```
export YDB_PASSWORD=password_123
ydbops restart --tenant \
  --endpoint grpc://<cluster-fqdn> \
  --availability-mode strong --verbose --hosts=7,8 \
  --user jorres --kubeconfig ~/.kube/config \
  --nodes-inflight 3 \
  --tenants-inflight 2
```

---

## For developers:

### Prerequisites

- Go 1.21
- `changie` tool for keeping a changelog

### How to build

Execute `make build-in-docker`, you will get binaries for Linux and MacOS, both amd and arm.

### How to run tests

Ginkgo testing library is used. Do:

```
ginkgo test -vvv ./tests
```

### How to develop

- develop a feature
- invoke `changie new` and complete a small interactive form. (Get changie from https://changie.dev )
- don't forget to changie-generated file to your PR into master branch

### How to release a new version

1. Invoke Github action `create-release-pr` job, it will create a PR with `CHANGELOG.md` containing all diffs 
2. After making sure that `CHANGELOG.md` looks nice, just merge the PR from step 1, and the commit into master
   will be automatically tagged, and a new release with new binaries will be automatically published!
