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
