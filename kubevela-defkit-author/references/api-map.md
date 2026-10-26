# defkit API map

Source: `github.com/oam-dev/kubevela/pkg/definition/defkit` at kubevela v1.11.0, read from
the package source on 2026-10-13. This is a map, not the reference: for any signature,
run `go doc github.com/oam-dev/kubevela/pkg/definition/defkit <Name>` against the version
in your module's `go.mod`. Names here were taken from the exported identifiers, not from
memory. If a name below does not exist in your version, trust `go doc`.

## Define a definition

| You want | Start with | Then |
|---|---|---|
| A component | `defkit.NewComponent(name)` | `.Description()`, `.Workload(apiVersion, kind)` or `.AutodetectWorkload()`, `.Params(...)`, `.Template(func(tpl *defkit.Template))`, `.HealthPolicy(expr)` or `.HealthPolicyExpr(...)`, `.CustomStatus(expr)`, `.Labels(map)`, `.Helper(name, param)`, `.ChildResourceKind(...)`, `.PodSpecPath(path)`, `.Version(v)` |
| A trait | `defkit.NewTrait(name)` | `.AppliesTo(workloads...)`, `.ConflictsWith(traits...)`, `.PodDisruptive(bool)`, `.Stage(stage)`, `.Params(...)`, `.Template(func(tpl *defkit.Template))` using `tpl.Patch()`, `.ManageWorkload()`, `.ControlPlaneOnly()`, `.RevisionEnabled()` |
| A policy | `defkit.NewPolicy(name)` | `.Description()`, `.Params(...)`, `.Helper(name, param)`, optional `.Template(func(pt *defkit.PolicyTemplate))` with `pt.Set(name, value)`, `.ManageHealthCheck()` |
| A workflow step | `defkit.NewWorkflowStep(name)` | `.Description()`, `.Category(c)`, `.WithImports("vela/builtin", ...)`, `.Params(...)`, `.Template(func(tpl *defkit.WorkflowStepTemplate))` |
| Registration | `defkit.Register(def)` in `init()` | `defkit.All()`, `defkit.Components()`, `defkit.Count()`, `defkit.Clear()` for tooling and tests |
| Output | `def.ToCue()`, `def.ToYAML()`, `def.ToParameterSchema()` | Generation is normally done by the module's `cmd/defkit generate`, not by hand |
| Escape hatch | `.RawCUE(cue)`, `.WithImports(...)`, trait `.TemplateBlock(cue)` | Only for CUE the builders cannot express; say why in a comment |
| Placement | `.RunOn(conditions...)`, `.NotRunOn(conditions...)` from the `placement` subpackage | Validated at `Register`; conflicting conditions panic |

## Parameters

Constructors: `String`, `Int`, `Bool`, `Float`, `Enum`, `Array`, `StringList`, `IntList`,
`List`, `Map`, `Object`, `StringKeyMap`, `DynamicMap`, `Struct`, `Field(name, ParamType)`,
`OneOf` with `Variant`, `ClosedUnion` with `ClosedStruct`, `ConditionalParams`.

Common modifiers on each: `.Required()`, `.Optional()`, `.Default(v)`, `.Description(s)`,
`.Short(s)`, `.Ignore()`. Type-specific constraints: `String` has `.Values(...)`,
`.OpenEnum()`, `.Pattern(regex)`, `.MinLen(n)`, `.MaxLen(n)`, `.NotEmpty()`; `Int` and
`Float` have `.Min(n)`, `.Max(n)`; `Array` has `.Of(ParamType)`, `.WithFields(...)`,
`.OfEnum(...)`, `.MinItems`, `.MaxItems`, `.NotEmpty`; `Map`/`Object` have `.Of(ParamType)`,
`.WithFields(...)`, `.OfObject(...)`, `.Closed()`, `.ConditionalFields(...)`; shared
struct shapes via `.WithSchemaRef(name)` paired with `def.Helper(name, param)`.

Parameters are also values and conditions inside templates: `.IsSet()`, `.NotSet()`,
`.Eq(v)`, `.Ne(v)`, `.Gt`, `.Gte`, `.Lt`, `.Lte`; `String` adds `.Contains`, `.Matches`,
`.StartsWith`, `.EndsWith`, `.In(...)`, `.Concat(s)`, `.Prepend(s)`; `Bool` adds
`.IsTrue()`, `.IsFalse()`; `Array` and `Map` add `.IsEmpty()`, `.IsNotEmpty()`, `.LenEq(n)`
and friends, `.Contains(v)`, `.HasKey(k)`; `Map` and `Struct` give `.Field(path)`.

Declare a parameter once with its constraints in the definition's `Params(...)`. Inside
the template function, re-declare by the same name with the bare constructor
(`defkit.String("image")`) to reference it; constraints are not repeated there. The
examples file shows this pattern.

## Templates and resources

- Component template: `tpl.Output(resource)` for the workload, `tpl.Outputs(name, resource)`
  for auxiliary outputs, `tpl.OutputsIf(cond, name, resource)`, `tpl.OutputsGroupIf(cond, fn)`.
- Build a resource with `defkit.NewResource(apiVersion, kind)` and chain `.Set(path, value)`,
  `.SetIf(cond, path, value)`, `.SpreadIf(cond, path, value)`, `.If(cond)` ... `.EndIf()`,
  `.ConditionalStruct(cond, path, fn)`, `.VersionIf(cond, apiVersion)` with
  `NewResourceWithConditionalVersion(kind)`, `.Directive(path, directive)`.
- Paths use dotted notation with brackets for keys and indexes:
  `spec.template.spec.containers[0].image`, `metadata.labels[app.oam.dev/name]`.
- Runtime context: `defkit.VelaCtx()` then `.Name()`, `.AppName()`, `.Namespace()` and the
  other accessors documented on the context type; `defkit.ContextOutput()` and
  `defkit.ContextOutputExists(path)` for reading rendered output in status expressions.
- Values and expressions: `defkit.Interpolation(parts...)`, `defkit.Format(fmt, args...)`,
  `defkit.ConvertString(v)`, `defkit.InlineArray(map)`, `defkit.ArrayConcat(a, b)`,
  comparisons `Eq`, `Ge`, `Gt`, `Le`, logical `And`, `AllConditions`, raw `CUEExpr(s)`.
- Typed objects: `defkit.FromTyped(runtime.Object)` exists, but the examples build output
  with `NewResource` and paths to avoid a dependency on Kubernetes Go types; the API
  version is still explicit, as the first argument of `NewResource`.

## Collections and helpers

- Iterate a list parameter: `defkit.Each(list).Map(FieldMap)` with `defkit.FieldRef(name)`,
  `defkit.Nested(FieldMap)`, `defkit.NestedFieldMap`, `defkit.Optional(name)`,
  `defkit.OptionalFieldRef(name)`, and `.MapVariant(discriminator, value, FieldMap)` for
  `OneOf` parameters. `defkit.From(source)`, `defkit.ForEachIn(source)`, `defkit.ForEachMap()`.
- Named template helpers: `tpl.Helper(name)` then `.FromFields(source, fields...)`,
  `.FromHelper(h)`, `.Pick(fields...)`, `.PickIf(cond, field)`, `.MapBySource(map)`,
  `.Dedupe(field)`, `.Build()`. The result is a `*HelperVar` usable as a value.
- Struct helpers: `defkit.HelperStruct(fields...)`, `defkit.HelperField(name, value)`,
  `defkit.HelperFieldIf(cond, name, value)`, `defkit.FromFields(source, fields...)`,
  `defkit.ConcatExpr(helper, fields...)`.
- Item-level conditions inside collections: `defkit.Item()`, `defkit.ItemFieldIsSet(field)`,
  `defkit.FieldEquals(field, v)`, `defkit.FieldExists(field)`, `defkit.LenEq(source, n)`,
  `defkit.LenGe(source, n)`, `defkit.HasExposedPorts(ports)`.

## Health and status

- Builders: `defkit.Health()` with `.IntField(name, path, default)`, `.HealthyWhen(exprs...)`,
  `.Build()`; presets `DeploymentHealth()`, `DaemonSetHealth()`, `JobHealth()`,
  `CronJobHealth()`; status presets `DeploymentStatus()`, `DaemonSetStatus()`.
- Expressions: `defkit.StatusEq(a, b)`, `defkit.StatusOr(...)`, `defkit.HealthPolicy(expr)`,
  `defkit.CustomStatusExpr(expr)`. Pass the built string to `.HealthPolicy()` or
  `.CustomStatus()` on the definition.

## Patch traits

- In a trait template, `tpl.Patch()` returns a `*PatchResource` with `.Set(path, value)`,
  `.SetIf(cond, path, value)`, `.PatchKey(path, key, elements...)` which emits
  `// +patchKey=key`, and `.PatchStrategyAnnotation(path, strategy)`.
- `tpl.PatchStrategy(strategy)` sets the strategy on the whole patch block; `retainKeys`
  and `replace` are the ones the merge engine recognises. `open` is emitted by one code
  path but is not implemented by the engine, so do not rely on it.
- Container-targeted patches: `defkit.PatchContainer`-style helpers in `patch_container.go`
  match containers by name and apply fields; read `go doc` for the current builder.

## Workflow steps

- Builtin delegation: `tpl.Builtin(name, "builtin.#Suspend").WithFullParameter().Build()`
  after `.WithImports("vela/builtin")`.
- Operations: `defkit.KubeRead(apiVersion, kind)`, `defkit.KubeApply(value)`,
  `defkit.HTTPPost(url)`, `defkit.WaitUntil(expr)`, `defkit.Fail(message)`,
  `defkit.ConvertString(v)`.

## Testing

- `def.Render(defkit.TestContext().WithName(n).WithParam(k, v)...)` returns a
  `*RenderedResource` with `.APIVersion()`, `.Kind()`, `.Get(path)`, `.Data()`;
  `def.RenderAll(...)` returns every output. `defkit.SkipCompile()` is a render option.
- Direct template execution: `tpl := defkit.NewTemplate(); def.GetTemplate()(tpl)` then
  `tpl.GetOutput()`, `tpl.GetOutputs()`, `tpl.GetPatch()`.
- Gomega matchers in `pkg/definition/defkit/testing/matchers`: `BeDeployment()`,
  `BeService()`, `BeConfigMap()`, `BeSecret()`, `BeIngress()`, `BeResourceOfKind(kind)`,
  `HaveAPIVersion(v)`, `HaveSetOp(path)`, `HaveOpCount(n)`, `HaveParamNamed(name)`,
  `BeRequired()`, `BeOptional()`, `HaveDefaultValue(v)`, `HaveDescription(s)`.
