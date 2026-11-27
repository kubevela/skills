# Runtime tests against an emulator

Source: the worked module in `assets/aws-dynamodb`, run on k3d with vela-core
v1.12.0-alpha.1, the ACK DynamoDB controller 1.16.0 and AWS Emulator 1.0.533
(`ghcr.io/kubevela/aws-emulator`), with `github.com/kubevela/cloud-emulators/testkit`.

This is the procedure behind step 5 of the skill for a definition that provisions an AWS
service. It ends with a suite anyone can rerun and a record that says, for every lever,
whether it was proven. Copy `assets/aws-dynamodb/test/emulator` and change the service
parts; do not write the setup again.

## What is proven, and for which provider

| Provider and provisioner | Status |
|---|---|
| AWS DynamoDB on ACK | Proven. `assets/aws-dynamodb` runs in CI against the AWS Emulator, and `testkit` has the helpers. |
| Other AWS services on ACK | Not proven yet. The same helpers and steps are expected to apply, but no module has been run. Each service needs its own run, starting with the fidelity report for its operations. |
| AWS on Crossplane | Not established. No module has been run: the endpoint is set in the `ProviderConfig`, and `testkit` has no check for it. |
| GCP on Config Connector or Crossplane | Not established. No emulator image, no `testkit` helpers and no worked module exist here, and whether the controller can be pointed at an emulator has not been tried. |
| Azure on Azure Service Operator or Crossplane | Not established, for the same reasons. |

For a provider or provisioner that is not established, do not invent an emulator setup
and call the result verified. Do the unit tests, the render and the server check as usual.
Then tell the user that runtime verification has no established path here, and ask
whether they have an emulator they want used on a disposable cluster, or whether runtime
verification waits. Until a lever has been read back from a service, its row in the
record is `untested`, and the blocker is "no established emulator path for <provider>".

Running against a real cloud account is not one of the options this procedure offers.
It needs a controller that manages real resources, which is exactly what the rules below
keep this suite away from. If the user wants that, it is a separate setup that they own
and authorize, described in `references/real-cloud-tests.md`; the suite and `testkit` on
this page do not apply to it, and its record is a second section of `docs/coverage.md`,
kept apart from this one because an emulator proves shapes and a real run proves timing,
orderability, reachability and delete behaviour.

If the user names an emulator and it works, three things on this page carry over
unchanged: where the suite may run, the rules the specs follow, and the record. The
setup does not: how the controller gets its endpoint, how to check it, and how to install
it are specific to AWS on ACK, and have to be worked out and written down for the new
provider. The result is worth a worked module of its own next to `assets/aws-dynamodb`.

## Where this runs, and where it never does

- **Only a disposable cluster: k3d or kind, on a laptop or created inside a CI job.** The
  suite refuses any other kubeconfig context.
- **Never a cluster whose controller manages real cloud resources.** A controller takes
  its endpoint when it is installed. Repointing one at an emulator and back means
  reinstalling it while it holds resources it believes it created elsewhere. Do not
  propose it, and do not change the endpoint of an installed controller.
- **One emulator per cluster, shared.** A controller has one endpoint, so every suite on
  the cluster uses the same emulator. Use the one that is running; deploy the pinned
  release only when there is none. Runs stay apart by namespace and name prefix, not by
  emulator.
- **The emulator has no access control.** It accepts any request. Test data only.

## Before writing a spec

1. **Check the cluster is one you may use.** `kubectl config current-context` starts with
   `k3d-` or `kind-`. Otherwise stop and ask for a disposable cluster.
2. **Check the provisioner and its endpoint.** `get_platform_info` with `service` and
   `kind` names the provisioner and, when the controller is configured with one, the
   `endpointOverride`. Three cases:
   - No controller: ask the user, then install it pointing at the emulator (below).
   - A controller whose `endpointOverride` is the emulator's Service URL: proceed.
   - A controller with no override, or another one: it talks to AWS or to something else.
     Stop. Do not create resources through it and do not reconfigure it.
   - An endpoint you cannot establish: the read was denied, several controller
     Deployments serve the service, or the endpoint is not a literal value on the
     Deployment. Report the endpoint as unknown and stop before creating anything. Unknown
     is not "probably the emulator".

   Without the MCP tools, read the `AWS_ENDPOINT_URL` environment variable of the
   controller's Deployment; an ACK chart sets it from `aws.endpoint_url`. The suite makes
   the same check itself with `RequireACKOnEmulator`, so it holds without an agent too.
3. **Install the controller for the emulator**, after the user agrees. The emulator
   goes first: an ACK controller calls STS when it starts, and without an emulator to
   answer it never becomes ready. Look for a running one before deploying anything
   (`kubectl -n aws-emulator get deployment,service aws-emulator`). If it is there, use
   it as it is. If it is not, apply the manifest of the
   [kubevela/cloud-emulators release](https://github.com/kubevela/cloud-emulators/releases)
   whose version the suite pins, and wait for its rollout. Then install the controller.
   Look the chart version up; do not guess it. The keys are made up: the emulator ignores
   them.

   ```sh
   kubectl create namespace ack-system
   kubectl -n ack-system create secret generic ack-emulator-credentials \
     --from-literal=credentials=$'[default]\naws_access_key_id = emulator-not-a-real-key\naws_secret_access_key = emulator-not-a-real-secret\n'
   helm install ack-<service> oci://public.ecr.aws/aws-controllers-k8s/<service>-chart --version <looked-up> -n ack-system \
     --set aws.region=us-west-2 \
     --set aws.endpoint_url=http://aws-emulator.aws-emulator.svc.cluster.local:4566 \
     --set aws.allow_unsafe_aws_endpoint_urls=true \
     --set aws.credentials.secretName=ack-emulator-credentials
   ```

   Then wait for the rollout. If the pod stays in `ImagePullBackOff` with a certificate
   error, the cluster's nodes cannot reach that registry; load the image into the cluster
   from the host (`docker pull <image>` then `k3d image import <image> -c <cluster>`, or
   `kind load docker-image`) and report that you did.
4. **Read the fidelity report for the operations the definition uses**, for the emulator
   version on the cluster:

   ```sh
   python3 scripts/emulator-fidelity.py --service dynamodb --version <running version> \
     --operations CreateTable,UpdateTable,UpdateTimeToLive
   ```

   An operation that does not pass there is a known limit of the endpoint. Mismatches
   named `...-rejected` or `...-precedence` are about the wording of errors for invalid
   input and do not limit a valid lever. Note what you found in the coverage record.

## The suite

Put it in `test/emulator/` behind the build tag `emulator`, so `go test ./...` stays
free of a cluster. Its setup uses `testkit` and nothing else for the cluster side:

| Step | Call |
|---|---|
| Connect, refusing anything but k3d or kind | `testkit.Connect()` |
| Emulator present, or deployed at a pinned version | `cluster.EnsureAWSEmulator(ctx, version)` |
| Controller installed, running and on the emulator | `cluster.RequireACKOnEmulator(ctx, "<service>")` |
| Install the definition the way a platform team does | `vela def apply-module <module> --conflict=overwrite` |
| A namespace and name prefix for this run | `cluster.NewRun(ctx, "<service>")` |
| An endpoint for the AWS SDK in the test process | `cluster.ForwardAWSEmulator(ctx)` |
| Delete what the run made, wait for the controller, delete the namespace | `cluster.EndRun(ctx, run, owners, dependents)` |

Rules the specs follow:

- **One spec per lever, or per group of levers that only make sense together**, such as
  a billing mode and its capacity. A lever without a spec is untested, and untested needs
  a blocker (see the record below).
- **Apply through KubeVela, read back through the service.** Apply an Application that
  uses the definition, then assert on the service's own API (`DescribeTable`,
  `GetBucketVersioning`) with the AWS SDK. A rendered object, an accepted server check and
  a `True` condition are not proof that the service has the setting.
- **Ready means synced and not terminal.** `testkit.ACKReady` reads `ACK.ResourceSynced`
  and `ACK.Terminal` together. A resource the controller gave up on keeps its last Synced
  value.
- **After an update, wait on the service, not on the condition.** ACK conditions carry no
  observedGeneration; a `True` read right after a change can answer the previous spec.
  Poll the service API until it shows the new value.
- **Application health lags an update.** The controller is back in sync within seconds,
  but KubeVela re-checks a component's health on its re-sync period, 5 minutes by default.
  Allow that long when a spec asserts the Application is healthy again after an update,
  and assert it in one spec, not in every one.
- **Cover the unhappy path.** One spec applies a spec the service rejects for good, and
  asserts `ACK.Terminal` is `True`, the component is unhealthy and nothing was created.
- **Cover delete.** One spec deletes the Application and asserts the resource is gone
  from the service; one asserts the retain policy keeps it, then removes it with the SDK.
- **Fixtures look made up and carry the run's prefix.** Names such as
  `<run id>-update`. A lever that would stop cleanup, such as deletion protection, is
  switched off again in `DeferCleanup`.
- **Supporting resources come from the SDK.** A KMS key a table needs is created through
  the emulator's API in the spec, not by another controller.

## The record

`docs/coverage.md` in the module carries a section with exactly this table:

```markdown
## Runtime verification

| Lever | Outcome | Evidence or reason |
|---|---|---|
| timeToLive | verified end to end | DescribeTimeToLive |
| warmThroughput | endpoint-limited | UpdateTable returns `ValidationException: ...` |
| tableReplicas | untested | needs a second region, and the emulator serves one |

Counts: 1 verified end to end, 1 endpoint-limited, 1 untested.
```

- Every parameter of the definition has a row.
- **verified end to end** names the API call that read the value back.
- **endpoint-limited** quotes the error or names the limitation. Find the smallest spec
  that shows it and keep it as a spec that asserts the refusal, with a comment saying
  what to do when it fails. A newer emulator that supports the lever then fails the spec
  instead of leaving a stale row.
- **untested** names what blocks the test: a second region, a controller that is not
  installed, an account-level setting. "Not exercised yet" is not a reason. If nothing
  blocks the lever, write the spec.

Check the record before delivering:

```sh
python3 scripts/check-coverage.py docs/coverage.md --cue <generated definition>.cue
```

It fails on a missing parameter, an outcome outside the three, a pass or a limit without
evidence, an untested row without a blocker, and counts that disagree with the table.
