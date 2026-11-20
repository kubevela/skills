# dynamodb-table: coverage and runtime verification

A definition for a cloud service narrows the service twice: the controller's CRD exposes
part of the API, and the definition exposes part of the CRD. This page shows both cuts and
what was checked against a running service.

## Coverage

| DynamoDB API lever | ACK CRD field (dynamodb-controller 1.16.0) | Definition parameter |
|---|---|---|
| TableName | spec.tableName | tableName (defaults to the component name) |
| AttributeDefinitions | spec.attributeDefinitions | attributeDefinitions |
| KeySchema | spec.keySchema | keySchema |
| BillingMode | spec.billingMode | billingMode |
| ProvisionedThroughput | spec.provisionedThroughput | provisionedThroughput |
| OnDemandThroughput | spec.onDemandThroughput | onDemandThroughput |
| WarmThroughput | spec.warmThroughput | warmThroughput |
| GlobalSecondaryIndexes: IndexName, KeySchema, Projection, ProvisionedThroughput, OnDemandThroughput, WarmThroughput | spec.globalSecondaryIndexes | globalSecondaryIndexes |
| LocalSecondaryIndexes: IndexName, KeySchema, Projection | spec.localSecondaryIndexes | localSecondaryIndexes |
| StreamSpecification | spec.streamSpecification | streamSpecification |
| SSESpecification: Enabled, SSEType, KMSMasterKeyId | spec.sseSpecification | sseSpecification |
| SSE with a key managed as a Kubernetes object | spec.sseSpecification.kmsMasterKeyRef | — not installed: needs the ACK KMS controller |
| TableClass | spec.tableClass | tableClass |
| DeletionProtectionEnabled | spec.deletionProtectionEnabled | deletionProtectionEnabled |
| ResourcePolicy | spec.resourcePolicy | resourcePolicy |
| Tags | spec.tags | tags |
| UpdateTimeToLive | spec.timeToLive | timeToLive |
| UpdateContinuousBackups (point-in-time recovery) | spec.continuousBackups | continuousBackups |
| UpdateContributorInsights | spec.contributorInsights | contributorInsights |
| Replicas (global tables) | spec.tableReplicas | — not exposed by this definition |
| GlobalTableSettingsReplicationMode, GlobalTableSourceArn | — not in CRD | — not reachable from KubeVela |
| VectorIndexes | — not in CRD | — not reachable from KubeVela |
| What happens to the table on delete | annotation `services.k8s.aws/deletion-policy` | deletionPolicy |

Sources, fetched 2026-11-20 from
<https://docs.aws.amazon.com/amazondynamodb/latest/APIReference/>: `API_CreateTable`
and the data types it uses (`AttributeDefinition`, `KeySchemaElement`,
`ProvisionedThroughput`, `OnDemandThroughput`, `GlobalSecondaryIndex`,
`LocalSecondaryIndex`, `Projection`, `StreamSpecification`, `SSESpecification`, `Tag`,
`WarmThroughput`), `API_UpdateTimeToLive` with `TimeToLiveSpecification`, and
`PointInTimeRecoverySpecification`. The limits in the schema come from these pages. The
other operations the controller calls (UpdateTable, UpdateContributorInsights,
PutResourcePolicy, TagResource, DeleteTable) were not read for limits; their behaviour is
recorded under Runtime verification. The CRD is the `Table` served by
`public.ecr.aws/aws-controllers-k8s/dynamodb-controller:1.16.0`.

Why the gaps exist:

- **tableReplicas, not exposed.** Left out by the author on 2026-11-20. Replicas live in
  other regions, and the test endpoint is a single-region emulator, so nothing about them
  could be verified. It is a schema edit away: ask for it.
- **kmsMasterKeyRef, not installed.** The field points at a `Key` object of the ACK KMS
  controller, which was not on the cluster. Use `sseSpecification.kmsMasterKeyID` with a
  key ID or ARN until that controller is installed.
- **Not in CRD.** These need a release of the ACK DynamoDB controller that adds them.

Decisions that are not gaps:

- **No defaults.** The API reference documents none for these fields, so the definition
  sets none and DynamoDB's behaviour applies.
- **billingMode is required**, although the API does not require it. Unset means
  PROVISIONED, which needs capacity numbers, so a table with neither can never be
  created; and how a table is billed is a cost decision for the author to state. With
  the mode always present, the rules between it and the throughput blocks are checked
  at render: PROVISIONED needs `provisionedThroughput` and takes no
  `onDemandThroughput`; PAY_PER_REQUEST takes no `provisionedThroughput`.
- **Projection and request-limit rules are checked at render** too: `nonKeyAttributes`
  goes with `INCLUDE` and only with it, and a request limit is at least 1 or -1.
- **What the schema does not check.** That every key attribute is declared in
  `attributeDefinitions`, that a key is one HASH optionally followed by one RANGE, and
  that an index's capacity matches the table's billing mode. These span list elements,
  and written as schema they would fail with errors harder to read than DynamoDB's own.
  The service rejects them, ACK sets `ACK.Terminal`, and the component is unhealthy; the
  suite proves that path.
- **A short component name needs `tableName`.** The table name defaults to the component
  name, and DynamoDB wants at least 3 characters.
- **Limits in the schema** come from the API reference: key and index name patterns,
  1 to 2 key elements, at most 20 global and 5 local indexes, capacity of at least 1,
  recovery period 1 to 35 days, at most 50 tags.

## Runtime verification

Each lever was applied through KubeVela and the ACK controller to the AWS Emulator
1.0.533 (`ghcr.io/kubevela/aws-emulator`) on a k3d cluster with vela-core 1.12.0-alpha.1,
and read back with the DynamoDB API. The suite in `test/emulator/` repeats this, one spec
per row or group of rows.

The emulator's fidelity report for 1.0.533 was read first for the operations the
controller calls. UpdateTimeToLive, UpdateContinuousBackups, UpdateContributorInsights,
PutResourcePolicy and DeleteTable pass every scenario. CreateTable, UpdateTable,
DescribeTable and TagResource have mismatches, all in scenarios about how invalid input
is rejected. The one limit found at runtime, the refusal of `onDemandThroughput` on a global
secondary index, is not in the report.

| Lever | Outcome | Evidence or reason |
|---|---|---|
| tableName, and the default from the component name | verified end to end | DescribeTable |
| attributeDefinitions, keySchema (simple and composite) | verified end to end | DescribeTable |
| billingMode PAY_PER_REQUEST | verified end to end | DescribeTable, BillingModeSummary |
| billingMode PROVISIONED with provisionedThroughput | verified end to end | DescribeTable, ProvisionedThroughput; the emulator returns no BillingModeSummary for a provisioned table |
| billingMode switched from PAY_PER_REQUEST to PROVISIONED | verified end to end | DescribeTable |
| provisionedThroughput changed on an existing table | verified end to end | DescribeTable |
| onDemandThroughput | verified end to end | DescribeTable |
| warmThroughput | verified end to end | DescribeTable |
| globalSecondaryIndexes: name, key, projection with nonKeyAttributes, provisionedThroughput | verified end to end | DescribeTable |
| globalSecondaryIndexes: warmThroughput | verified end to end | DescribeTable |
| globalSecondaryIndexes: onDemandThroughput | endpoint-limited | CreateTable returns `NotImplemented: operation CreateTable with GlobalSecondaryIndexes carrying OnDemandThroughput is scaffolded but not implemented in this emulator`; a spec pins the refusal |
| globalSecondaryIndexes added to and removed from an existing table | verified end to end | DescribeTable |
| localSecondaryIndexes: name, key, projection | verified end to end | DescribeTable |
| streamSpecification | verified end to end | DescribeTable |
| sseSpecification enabled with sseType KMS | verified end to end | DescribeTable, SSEDescription |
| sseSpecification kmsMasterKeyID | verified end to end | DescribeTable, SSEDescription.KMSMasterKeyArn equals a key made with KMS CreateKey |
| tableClass STANDARD and STANDARD_INFREQUENT_ACCESS | verified end to end | DescribeTable |
| timeToLive | verified end to end | DescribeTimeToLive |
| continuousBackups with recoveryPeriodInDays | verified end to end | DescribeContinuousBackups |
| contributorInsights | verified end to end | DescribeContributorInsights |
| deletionProtectionEnabled, switched on and off again | verified end to end | DescribeTable; DeleteTable is refused while it is on |
| resourcePolicy | verified end to end | GetResourcePolicy |
| tags, at creation and changed later | verified end to end | ListTagsOfResource |
| deletionPolicy unset: the table is deleted with the Application | verified end to end | DescribeTable returns ResourceNotFound |
| deletionPolicy retain: the table stays | verified end to end | DescribeTable |
| Health: a table DynamoDB rejects for good is unhealthy | verified end to end | ACK.Terminal is True and DescribeTable returns ResourceNotFound |

Counts: 25 verified end to end, 1 endpoint-limited, 0 untested.

## What consumers should know

- **Quote `attributeType`.** YAML reads a bare `N` as `false`, and the Application is
  rejected with a type error.
- **Health lags after a change.** ACK goes out of sync and back within seconds of an
  update, but KubeVela looks at the component again on its re-sync period (5 minutes by
  default). After changing a healthy table, the Application can show `unhealthy` for up
  to that long while the table is already updated.
- **An invalid key is caught late.** The schema cannot check that every key attribute is
  declared in `attributeDefinitions`. DynamoDB rejects it, ACK sets `ACK.Terminal`, and
  the component stays unhealthy until the Application is fixed.
