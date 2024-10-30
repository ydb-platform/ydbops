# ydbops

`ydbops` utility is used to perform various ad-hoc and maintenance operations on YDB clusters.

For comprehensive documentation, refer to [ydb.tech](https://ydb.tech/docs/en/reference/ydbops/)

## Prerequisites:

1. Go 1.21

## How to build

Execute `make build-in-docker`, you will get binaries for Linux and MacOS, both amd and arm.

## How to run tests

Ginkgo testing library is used. Do:

```
ginkgo test -vvv ./tests
```

## How to use:

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

## How to create a new version
1. Define new semver version number (e.g. 1.1.0)
2. Update CHANGELOG.md with proper information about new version. Add the following to the beginning of the file, before previous entries:
```
## <your-version-number>

- change 1
- change 2
```
3. Create a pull request, wait for it to be merged
4. Github actions will create both the tag and the release with binaries for all platforms
