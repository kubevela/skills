# Worked examples

Source: `github.com/kubevela/vela-go-definitions` at commit `8b762d0635bb` (2026-08-21),
which pins `github.com/oam-dev/kubevela v1.10.5-0.20260524210911-a24d3a9c644f`. That
repository holds the built-in component, trait, policy and workflow-step definitions
written with defkit, each with tests. Read the file closest to what you are building
before writing anything; copy its structure, then change the parameters and template.

| You are writing | Closest example | Why |
|---|---|---|
| A workload component | `components/worker.go`, then `components/webservice.go` | Deployment output, probes, volumes, env, image pull secrets, health policy |
| A batch component | `components/task.go`, `components/crontask.go` | Job and CronJob outputs |
| A simple patch trait | `traits/scaler.go` | One parameter, `retainKeys`, `Patch().Set` |
| A container-targeted trait | `traits/env.go`, `traits/sidecar.go`, `traits/resource.go` | Patch by container name |
| A trait that adds a resource | `traits/expose.go`, `traits/gateway.go` | Traits with `Outputs` |
| A policy with rules | `policies/apply_once.go`, `policies/topology.go` | Helpers, `WithSchemaRef`, struct fields |
| A workflow step delegating to a builtin | `workflowsteps/suspend.go`, `workflowsteps/deploy.go` | `Builtin(...).WithFullParameter()` |
| A step with HTTP or kube operations | `workflowsteps/request.go`, `workflowsteps/read_object.go` | `HTTPPost`, `KubeRead`, `KubeApply` |
| Shared parameter shapes | `components/shared_helpers.go`, `components/schemas.go` | Reusable helpers and schema refs |

## Module layout the examples use

```
components/   traits/   policies/   workflowsteps/     one Go file per definition, plus *_test.go
cmd/defkit/main.go        `defkit generate --output-dir vela-templates/definitions`; imports every package for init()
cmd/register/main.go      registry entry point used by `vela def apply-module` (fast path)
module.yaml               DefinitionModule metadata: name, maintainers, categories, placement
vela-templates/definitions/   generated CUE, never edited by hand
Makefile                  generate, test-unit (ginkgo), test-e2e (k3d), reviewable
```

Every definition ends with `func init() { defkit.Register(X()) }`.

## Component: worker (abridged from components/worker.go)

```go
func Worker() *defkit.ComponentDefinition {
    image := defkit.String("image").Description("Which image would you like to use for your service").Short("i")
    imagePullPolicy := defkit.String("imagePullPolicy").Optional().Description("Specify image pull policy for your service")
    cmd := defkit.StringList("cmd").Optional().Description("Commands to run in the container")
    env := defkit.List("env").Optional().Description("Define arguments by using environment variables").
        WithFields(
            defkit.String("name").Description("Environment variable name"),
            defkit.String("value").Optional().Description("The value of the environment variable"),
        )
    cpu := defkit.String("cpu").Optional().Description("Number of CPU units for the service, like `0.5` or `1`")
    memory := defkit.String("memory").Optional().Description("Memory required for the container")
    livenessProbe := defkit.Object("livenessProbe").Optional().
        Description("Instructions for assessing whether the container is alive.").
        WithSchemaRef("HealthProbe")

    return defkit.NewComponent("worker").
        Description("Long-running, scalable, containerized services running at backend without a network endpoint.").
        Workload("apps/v1", "Deployment").
        Labels(map[string]string{"ui-hidden": "true"}).
        CustomStatus(defkit.DeploymentStatus().Build()).
        HealthPolicy(defkit.Health().
            IntField("ready.updatedReplicas", "status.updatedReplicas", 0).
            IntField("ready.readyReplicas", "status.readyReplicas", 0).
            IntField("ready.replicas", "status.replicas", 0).
            IntField("ready.observedGeneration", "status.observedGeneration", 0).
            HealthyWhen(
                defkit.StatusEq("context.output.spec.replicas", "ready.readyReplicas"),
                defkit.StatusEq("context.output.spec.replicas", "ready.updatedReplicas"),
                defkit.StatusEq("context.output.spec.replicas", "ready.replicas"),
                defkit.StatusOr(defkit.StatusEq("ready.observedGeneration", "context.output.metadata.generation"),
                    "ready.observedGeneration > context.output.metadata.generation"),
            ).Build()).
        Params(image, imagePullPolicy, cmd, env, cpu, memory, livenessProbe).
        Helper("HealthProbe", workerHealthProbeParam()).
        Template(workerTemplate)
}

func workerTemplate(tpl *defkit.Template) {
    vela := defkit.VelaCtx()
    image := defkit.String("image")          // re-declared by name only; constraints live in Params above
    cmd := defkit.StringList("cmd")
    env := defkit.List("env")
    cpu := defkit.String("cpu")
    memory := defkit.String("memory")
    livenessProbe := defkit.Object("livenessProbe")
    imagePullPolicy := defkit.String("imagePullPolicy")

    deployment := defkit.NewResource("apps/v1", "Deployment").
        Set("spec.selector.matchLabels[app.oam.dev/component]", vela.Name()).
        Set("spec.template.metadata.labels[app.oam.dev/name]", vela.AppName()).
        Set("spec.template.metadata.labels[app.oam.dev/component]", vela.Name()).
        Set("spec.template.spec.containers[0].name", vela.Name()).
        Set("spec.template.spec.containers[0].image", image).
        SetIf(imagePullPolicy.IsSet(), "spec.template.spec.containers[0].imagePullPolicy", imagePullPolicy).
        SetIf(cmd.IsSet(), "spec.template.spec.containers[0].command", cmd).
        SetIf(env.IsSet(), "spec.template.spec.containers[0].env", env).
        SetIf(cpu.IsSet(), "spec.template.spec.containers[0].resources.limits.cpu", cpu).
        SetIf(cpu.IsSet(), "spec.template.spec.containers[0].resources.requests.cpu", cpu).
        SetIf(memory.IsSet(), "spec.template.spec.containers[0].resources.limits.memory", memory).
        SetIf(memory.IsSet(), "spec.template.spec.containers[0].resources.requests.memory", memory).
        SetIf(livenessProbe.IsSet(), "spec.template.spec.containers[0].livenessProbe", livenessProbe)

    tpl.Output(deployment)
}

func init() { defkit.Register(Worker()) }
```

The full file also shows: a `OneOf("type")` parameter with `Variant`s for volume types,
`defkit.Each(volumes).Map(...).MapVariant("type", "pvc", ...)` to render them,
`tpl.Helper(...).FromFields(...).Pick(...).Dedupe("name").Build()` for volume lists, and
`If(cond) ... EndIf()` blocks on the resource.

## Trait: scaler (traits/scaler.go, complete)

```go
func Scaler() *defkit.TraitDefinition {
    replicas := defkit.Int("replicas").Default(1).Description("Specify the number of workload")

    return defkit.NewTrait("scaler").
        Description("Manually scale K8s pod for your workload which follows the pod spec in path 'spec.template'.").
        AppliesTo("deployments.apps", "statefulsets.apps").
        PodDisruptive(false).
        Params(replicas).
        Template(func(tpl *defkit.Template) {
            tpl.PatchStrategy("retainKeys").
                Patch().Set("spec.replicas", replicas)
        })
}

func init() { defkit.Register(Scaler()) }
```

## Policy: apply-once (policies/apply_once.go, abridged)

```go
func ApplyOnce() *defkit.PolicyDefinition {
    applyOnceStrategy := defkit.Struct("strategy").WithFields(
        defkit.Field("affect", defkit.ParamTypeString).Description("When the strategy takes effect, e.g. onUpdate, onStateKeep").Optional(),
        defkit.Field("path", defkit.ParamTypeArray).Of(defkit.ParamTypeString).Description("Paths that allow configuration drift"),
    )
    applyOncePolicyRule := defkit.Struct("rule").WithFields(
        defkit.Field("selector", defkit.ParamTypeStruct).Optional().WithSchemaRef("ResourcePolicyRuleSelector"),
        defkit.Field("strategy", defkit.ParamTypeStruct).WithSchemaRef("ApplyOnceStrategy"),
    )
    return defkit.NewPolicy("apply-once").
        Description("Allow configuration drift for applied resources.").
        Helper("ApplyOnceStrategy", applyOnceStrategy).
        Helper("ApplyOncePolicyRule", applyOncePolicyRule).
        Params(
            defkit.Bool("enable").Description("Whether to enable apply-once for the whole application").Default(false),
            defkit.Array("rules").Description("Rules for configuring apply-once at resource level").WithSchemaRef("ApplyOncePolicyRule").Optional(),
        )
}
```

## Workflow step: suspend (workflowsteps/suspend.go, complete)

```go
func Suspend() *defkit.WorkflowStepDefinition {
    duration := defkit.String("duration").Optional().Description("Wait duration to resume workflow such as \"30s\", \"1min\" or \"2m15s\"")
    message := defkit.String("message").Optional().Description("The suspend message to show")

    return defkit.NewWorkflowStep("suspend").
        Description("Suspend the current workflow, it can be resumed by 'vela workflow resume' command.").
        Category("Process Control").
        WithImports("vela/builtin").
        Params(duration, message).
        Template(func(tpl *defkit.WorkflowStepTemplate) {
            tpl.Builtin("suspend", "builtin.#Suspend").
                WithFullParameter().
                Build()
        })
}

func init() { defkit.Register(Suspend()) }
```

## Test pattern (components/worker_test.go, abridged)

```go
import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    "github.com/oam-dev/kubevela/pkg/definition/defkit"
    . "github.com/oam-dev/kubevela/pkg/definition/defkit/testing/matchers"
)

var _ = Describe("Worker Component", func() {
    It("has the expected shape", func() {
        comp := components.Worker()
        Expect(comp.GetName()).To(Equal("worker"))
        Expect(comp.GetWorkload().Kind()).To(Equal("Deployment"))
        Expect(comp).To(HaveParamNamed("image"))
        Expect(comp).NotTo(HaveParamNamed("ports"))
    })

    It("produces a Deployment and nothing else", func() {
        tpl := defkit.NewTemplate()
        components.Worker().GetTemplate()(tpl)
        Expect(tpl.GetOutput()).To(BeDeployment())
        Expect(tpl.GetOutputs()).To(BeEmpty())
    })

    It("renders a minimal worker", func() {
        rendered := components.Worker().Render(
            defkit.TestContext().WithName("my-worker").WithParam("image", "busybox:latest"))
        Expect(rendered.Kind()).To(Equal("Deployment"))
        Expect(rendered.Get("metadata.name")).To(Equal("my-worker"))
        Expect(rendered.Get("spec.template.spec.containers[0].image")).To(Equal("busybox:latest"))
    })
})
```

Cover in your tests: defaults applied when a parameter is omitted, a boundary value for
each constrained parameter, an invalid value that must be rejected, and the health and
status expressions where the definition declares them.
