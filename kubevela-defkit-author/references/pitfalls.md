# Pitfalls

Each of these has been seen in review or follows directly from how defkit works.

1. **Inventing builder names.** The API map is a map. Before using any method, confirm it
   with `go doc github.com/oam-dev/kubevela/pkg/definition/defkit <Name>` against the
   version in `go.mod`. If it does not exist, use the nearest example's approach or the
   `RawCUE` escape hatch with a comment, and say so in the pull request.
2. **Repeating constraints in the template.** Declare `defkit.String("image").Required()...`
   once in `Params(...)`. In the template function, re-declare with the bare constructor
   `defkit.String("image")` to reference it. Constraints repeated in the template are
   ignored at best and confusing at worst.
3. **Hand-editing generated CUE.** `vela-templates/definitions/` is output. Change the Go,
   run `make generate`, and let `make check-diff` or CI catch a stale file.
4. **Pinning Kubernetes API types.** Build output with `NewResource(apiVersion, kind)` and
   paths. `FromTyped` exists but couples the definition to a client-go version, which the
   design deliberately avoids.
5. **Forgetting `Register`.** Without `func init() { defkit.Register(X()) }` the module's
   generator and `vela def apply-module` never see the definition. Tests can still pass.
6. **Assuming `patchStrategy=open` appends.** One code path emits it and a built-in CUE trait
   uses it, but the merge engine recognises `retainKeys`, `replace`, `jsonPatch` and
   `jsonMergePatch` only. For list merges use `PatchKey` on a field the elements share; for
   lists whose elements share no key, see kubevela issue 2939 before relying on any
   behaviour.
7. **Treating an offline render as validation.** `vela dry-run --offline` and
   `def.Render(TestContext())` prove the template produces objects. Application validation,
   admission and runtime health are not checked. Report the status block from the skill.
8. **Testing only the happy path.** The examples' tests check shape, outputs and one minimal
   render. Add defaults, boundaries, an invalid input, and the health expression.
9. **Schema refs without helpers.** `WithSchemaRef("HealthProbe")` needs a matching
   `.Helper("HealthProbe", param)` on the definition, or the generated CUE references an
   undefined type and `vela def vet` fails.
10. **Version drift between module and cluster.** The module's `go.mod` pins a kubevela
    version; the generated CUE targets the controller features of that version. Check
    `get_platform_info` or `vela version` against the pin before promising a feature such
    as definition versioning or placement constraints will work on the target cluster.
11. **Bare YAML values that are not strings.** YAML reads an unquoted `N`, `Y`, `yes`,
    `no`, `on` or `off` as a boolean, so `attributeType: N` reaches the definition as
    `false` and is rejected with "mismatched types bool and string". Quote such values in
    every sample, and say so in the parameter's usage string when an enum has one.
12. **Installing a definition by another route than the CLI.** Install a module the way a platform team
    does, with `vela def apply-module <dir>`, in tests too. A second apply of a changed
    definition needs `--conflict=overwrite`.
13. **Reading health right after an update.** A cloud controller is out of sync and back
    within seconds, but KubeVela re-checks a component on its re-sync period, 5 minutes by
    default. Until then the Application can show the health it had during the update.
    Prove a change with the service's API, and give an Application-health assertion after
    an update that much time.
14. **A reason that is not one.** "Untested: not exercised yet" in the coverage record
    means a spec is missing. Untested is for a lever something blocks, and the row names
    the blocker. `scripts/check-coverage.py` rejects placeholders such as "not exercised
    yet", "later" or "TODO"; whether another reason is a real blocker is for you and the
    reviewer to judge.
15. **A constraint the schema cannot express.** Some rules span parameters, such as every
    key attribute being declared in `attributeDefinitions`. The service rejects the
    request, the controller sets its terminal condition, and the component must then be
    unhealthy. Test that path once per definition; a health policy that reads only the
    synced condition reports such a resource as ready.
16. **Rules on a shape registered with `Helper`.** A helper type carries the fields of
    its shape but not its `Validators`. A shape with rules between its fields, such as a
    projection whose `nonKeyAttributes` goes only with `INCLUDE`, is written into each
    parameter that uses it (one Go function, called at each use) so the rules are
    emitted. Check the generated CUE for the rule's message at every use.
17. **Installing the controller before the emulator.** An ACK controller calls STS when
    it starts. Pointed at an emulator that is not there yet, it restarts until one
    answers. On a test cluster the emulator is deployed first.
18. **A health rule that reads only the primary output.** A component that emits a
    cluster and its instances, a pool and its nodes, or any container plus members, is
    reported healthy as soon as the container is, minutes before the members and the
    address a consumer needs exist. KubeVela then releases every component that
    `dependsOn` it. Emulators hide this because they complete everything at once; a real
    cloud does not. Keep the primary rule, including any endpoint-present check, and
    `And` it with `defkit.Health().Every(defkit.OutputsWithPrefix(...)).AllowEmpty()` per
    name prefix, `AllowEmpty` because a prefix may match no output yet, or `.At(ref)`
    per named output; `Every` judges only the outputs it selects. The in-process
    renderer does not evaluate `Every`: assert the CUE from `GetHealthPolicy()`.
