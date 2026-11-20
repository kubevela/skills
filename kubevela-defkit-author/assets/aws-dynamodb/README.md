# aws-dynamodb

A KubeVela definition module with one component, `dynamodb-table`: an Amazon DynamoDB
table provisioned through the
[ACK DynamoDB controller](https://github.com/aws-controllers-k8s/dynamodb-controller).
The definition is written in Go with defkit.

It is the worked example of the `kubevela-defkit-author` skill: copy its layout, its
tests and its emulator suite when authoring a definition for another cloud service.

```yaml
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  name: sessions
spec:
  components:
    - name: sessions
      type: dynamodb-table
      properties:
        attributeDefinitions:
          - attributeName: sessionId
            attributeType: "S"
        keySchema:
          - attributeName: sessionId
            keyType: HASH
        billingMode: PAY_PER_REQUEST
```

More samples are in [`examples/`](examples). Quote `attributeType` in YAML: a bare `N` is
read as the boolean `false`.

## What it needs

- KubeVela 1.11 or later.
- The ACK DynamoDB controller. The parameters follow its `Table` CRD at release 1.16.0.
  Region and credentials come from the controller's configuration, not from parameters.

## Parameters

Every parameter has the name and shape of the `Table` field it sets. Three are required:
`attributeDefinitions`, `keySchema` and `billingMode`. `tableName` defaults to the
component name; set it when that name is shorter than 3 characters, the shortest a table
name may be. No other parameter has a default: DynamoDB's own defaults apply.

One parameter is not a `Table` spec field: `deletionPolicy` (`delete` or `retain`) sets
ACK's `services.k8s.aws/deletion-policy` annotation.

Rules that span parameters are checked when the Application renders, with a message that
names the rule:

- `PROVISIONED` needs `provisionedThroughput` and takes no `onDemandThroughput`;
  `PAY_PER_REQUEST` takes no `provisionedThroughput`.
- A projection has `nonKeyAttributes` with `projectionType: INCLUDE`, and only then.
- A request limit is at least 1, or -1 to remove it.

The component is healthy when ACK reports `ACK.ResourceSynced` as `True` and
`ACK.Terminal` is not `True`. Its status message carries the table status and ARN.

What the definition covers, what it leaves out and why, and what was checked against a
running service is in [`docs/coverage.md`](docs/coverage.md). Check the record after changing
a parameter or a spec:

```shell
python3 ../../scripts/check-coverage.py docs/coverage.md --cue cue-generated/components/dynamodb-table.cue
```

## Use it

```shell
vela def validate-module .
vela def apply-module . --dry-run     # preview
vela def apply-module .               # your own cluster
```

## Tests

```shell
go test ./...                                    # schema and render, no cluster
go test -tags emulator ./test/emulator/...       # against the AWS Emulator
```

The second suite applies Applications to a cluster and reads every setting back with the
DynamoDB API. It only runs on a disposable **k3d or kind** cluster and refuses any other
context. It needs:

- KubeVela on the cluster, and the `vela` CLI (on `PATH`, or `VELA_BIN`).
- An ACK DynamoDB controller installed with the emulator as its endpoint. The suite
  refuses to run when a DynamoDB controller on the cluster has a different endpoint or
  none, since that controller talks to AWS. Deploy the emulator before the controller:
  the controller calls it when it starts and does not come up without it.

  ```shell
  helm install ack-dynamodb oci://public.ecr.aws/aws-controllers-k8s/dynamodb-chart \
    --version 1.16.0 -n ack-system --create-namespace \
    --set aws.region=us-west-2 \
    --set aws.endpoint_url=http://aws-emulator.aws-emulator.svc.cluster.local:4566 \
    --set aws.allow_unsafe_aws_endpoint_urls=true \
    --set aws.credentials.secretName=<a Secret with any keys>
  ```

The cluster setup and its safety checks come from
[`testkit`](https://github.com/kubevela/cloud-emulators/tree/main/testkit). The KubeVela
helpers are in `test/emulator/vela_test.go`, which knows nothing about DynamoDB and can be
copied as it is.

The suite deploys the [AWS Emulator](https://github.com/kubevela/cloud-emulators) at a
pinned version when the cluster has none, and uses the one that is there otherwise. One
emulator serves the whole cluster, because the controller has one endpoint. Each run
creates its own namespace and table-name prefix and deletes both at the end, so two runs
do not collide. The emulator, the controller and the definition stay installed.

## Layout

```
components/        the definition and its unit tests
examples/          sample Applications
test/emulator/     the suite that runs against the AWS Emulator
docs/coverage.md   API coverage and the runtime verification record
cue-generated/     output of `vela def gen-module`; do not edit
```
