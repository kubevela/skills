# Runtime tests against a real cloud account

Source: an `aurora-postgres` module (ACK rds-controller 1.14.2, Aurora PostgreSQL
Serverless v2, us-east-1) run from a dedicated k3d cluster with vela-core v1.12.0-alpha.1
on 2026-11-25, and the Redshift `Cluster` resource of aws-controllers-k8s/redshift-controller#6
set up the same day.

An emulator proves request and response shapes. It does not prove timing, DNS, what an
account may order, engine version floors, quotas, network reachability or what happens on
delete. Those only show up against the real service. A lever that passed the emulator
suite therefore earns a second, separate state here, and the record keeps the two apart.

This page is the setup `references/emulator-tests.md` points at when the user wants a
real-cloud run. The user owns and authorizes it: the account, the credentials, the budget
and the go-ahead for every resource that bills. Nothing here runs by default.

## Where it runs

- **A controller instance of its own.** A provisioner instance is one endpoint and one
  identity, and changing where it points changes what every resource it already holds
  will do. Never re-point an installed controller: on a cluster whose controllers serve an
  emulator or another account, a second cluster, or a namespace-scoped second controller
  kept apart by a watch selector, is where the real-cloud controller goes. Inventory what
  an instance owns before touching it; the platform-engineer skill has the procedure.
- **Credentials that can be revoked and never appear in output.** A Secret built through a
  pipe from the CLI profile (`--from-file=credentials=<(printf ...)`), a key or session
  the user deactivates when the test is over, and a spend alarm on the account before the
  first apply.
- **A disposable kube context.** The suite refuses any context that is not `k3d-` or
  `kind-`, as the emulator suite does; the cloud side is real, the cluster side is not.

## Before writing a spec

1. **Pick the cheapest shape that still exercises the lever**, and check it is orderable
   where the test runs: the smallest SKU, a scale-to-zero or serverless mode, one member,
   the shortest retention. Query the service, do not assume. On 2026-11-25 `dc2.large`
   was not orderable in the test account and scaling Aurora Serverless v2 to zero needed
   PostgreSQL 16.3 or newer. Pin what the query returned and write the date of the lookup
   next to the pin, because defaults move.
2. **Settle delete before create.** Every provider protects data on delete somewhere:
   final snapshots, deletion protection, retention locks, soft delete. Read how the
   provisioner exposes the choice (an annotation, `deletionPolicy`, a spec field) and its
   default, put it in the sample, and give the coverage table a row for what an
   Application delete does. A resource you cannot delete is the most expensive lever.
3. **Name every fixture.** Whatever the Application depends on but does not own, a
   firewall rule for the operator's egress address, a public endpoint because the consumer
   runs outside the provider's network, a pre-existing Secret, a project binding, is a
   fixture with an owner, the exact command that creates it and the one that removes it.
   The sample carries a placeholder for it that cannot be mistaken for a real value.
4. **Estimate the cost of one run** and write it in the module README: the metered rate
   of the shape, the expected wall time, the idle cost if teardown fails.

## The suite

Put it in `test/integration/` and gate it on an environment variable the user sets on
purpose, so `go test ./...` never bills anyone. The rules of the emulator suite carry
over: Ginkgo specs, one per lever or group, apply through KubeVela, read back through the
service's own API, fixtures that look made up. What changes:

- **Teardown is unconditional and waits for the provider.** `AfterSuite` deletes the
  Application, waits until the provisioner's resource is gone from the cluster, then polls
  the service until it reports the resource does not exist (`DBClusterNotFoundFault` and
  its equivalents), on failure too. A deleted CR with the cloud resource still billing is
  the failure mode to design against.
- **Health is proven by a consumer, not only by conditions.** One spec runs a client from
  inside the cluster against what the component exported, a `SELECT 1`, a GET on the
  service's own endpoint, and reads the service's view of that session where the service
  offers one. That is where ordering gaps appear: a resource reported ready before its
  members, an endpoint that does not resolve yet.
- **Timings are evidence.** Record create-to-ready, first-use latency on a scaled-to-zero
  shape, and delete-to-gone, from the provider's timestamps.
- **The run leaves a log outside the module**, never credentials, and the record cites
  the date and the account's region.

## The record

`docs/coverage.md` keeps the emulator table under `## Runtime verification` as the
complete, per-parameter record. A real-cloud run adds a second section with the same
format and the same three outcomes:

```markdown
## Real-cloud verification

Run on <date> against <provider, region> through <provisioner:version>.

| Lever | Outcome | Evidence or reason |
|---|---|---|
| engineVersion | verified end to end | DescribeDBClusters |
| connectionSecret | verified end to end | Secret keys read, SELECT 1 over the exported endpoint |
| dc2.large node type | endpoint-limited | not orderable in the account: DescribeOrderableClusterOptions lists ra3.large only |
| multiAZ | untested | one extra instance-hour per run; not in the budget |

Counts: 2 verified end to end, 1 endpoint-limited, 1 untested.
```

- Only levers the run exercised get rows; the section is a subset, and the emulator table
  stays the one `--cue` completeness is checked against.
- **endpoint-limited** here means the account or region refused it: not orderable, quota,
  unsupported version. Quote the refusal.
- **untested** names the blocker, which may be cost. "Not in the budget" with the number
  is a reason; "not exercised yet" is not.
- Include a row for the delete path and one for each consumer-side proof.

`scripts/check-coverage.py` validates both sections when the second is present.

## The server check on a shared cluster

Step 8 of the skill needs a namespace to run `kubectl apply --dry-run=server`. Create one
with a name nothing else would use, `<module>-check-<random>`, and delete only that. A
sample's namespace name may already exist on a shared cluster with someone's work in it;
an apply that prints `unchanged` for the namespace is the signal to stop.
