# Agent instructions

## The bar

For everything the provider manages, the plan shows every difference between the configuration and
the tenant, and the apply leaves the tenant matching the configuration.

- The same configuration gives the same tenant in every environment.
- A plan straight after an apply shows `No changes`.
- The configuration is the single source of truth. Never suggest `ignore_changes`, a change in the
  portal, or editing state by hand as a fix - not in code, error messages, docs or PR descriptions.
- A change that hides a difference, or leaves one behind in the tenant, is a bug even when every
  test passes.

Before finishing a change to a resource, answer these for every attribute it touches:

1. The configuration leaves it out. What does the plan show?
2. Someone changes it outside Terraform. Does the next plan show it?
3. The configuration stops setting it. Does the apply clear it in the tenant, and is the next plan
   clean?
4. The resource is imported with `-generate-config-out`. Is the generated configuration valid, and
   does it plan clean?

If an answer is "nothing shows" or "the tenant keeps it", fix that or list it under Known gaps.

## Check the API before relying on it

Most bugs found in this provider came from an assumption about the Management API that nobody
checked. The API's code is not in this repository, and a stub test only repeats the assumption.

Check each of these against the API's code or a dev tenant before writing code that depends on it,
and say in the PR which one you used:

- Whether an update merges or replaces an object or a list. Credential blocks are merged, so a
  member the request leaves out is kept.
- Whether a field accepts `null`. Some reject it and need `[]`, an empty string or `false` instead.
- What a cleared field looks like when read back: absent, `null`, `""` or `[]`.
- What a fresh tenant returns for a field that was never set.
- Whether a delete removes the object. Rules are soft-deleted: GET still returns 200, with
  `deletedAt` set.
- What the API clears by itself when a provider changes, and what it keeps.
- What another client can store that the provider can't express.

When the API can't do what the provider needs, say so and stop. Don't work around it in a way that
leaves the tenant different from the configuration.

## Schema rules

Pick the attribute's shape from this list. The schema tests in
`authenticator_configuration_schema_test.go` and `schema_test.go` pin most of these, so add a new
resource to the lists those tests use.

- **A value only the server sets**: Computed only. Add `UseStateForUnknown` when an update can't
  change it, or every update shows `(known after apply)`.
- **A scalar, list or set the API returns**: Optional and Computed. It keeps the tenant's value when
  the configuration leaves it out, so the unmanaged settings check below has to cover it.
- **A nested attribute (an object or a list of objects)**: Optional only, never Computed. Terraform
  plans an Optional and Computed nested attribute as unknown whenever the configuration leaves it
  out. That gives a change on every plan, and the value is never cleared. Read always fills it, and
  removing it from the configuration sends an explicit `null`.
- **A secret**: write-only, never Computed, never in state. Its credential block has a `_version`
  attribute, and the secret is only sent when that version changes.
- **A schema default**: only when it equals what the API does with a blank value. Adding or
  changing one changes existing tenants, so it needs a major release.

Two more rules:

- Don't accept a value that Read can't return. If the API reads `[]` back as absent, reject `[]` in
  validation and say to leave the attribute out. Otherwise the plan never settles.
- Terraform drops a key that a nested attribute doesn't define before the provider sees it. The
  provider can't reject a misspelt key, so don't write a test that expects it to.
- Changing an attribute's type or nesting needs a schema version bump and a state upgrader. Without
  them, state written by the previous release fails to load.

## Settings the configuration leaves out

An Optional and Computed setting with no schema default keeps the tenant's value when the
configuration leaves it out. `checkUnmanagedSettings` fails the plan for such a setting, unless the
tenant's value is blank or equals the default the Management API reports in `defaults`.

- Every resource with a setting like this calls the check from `ModifyPlan`.
  `TestEverySettingTheConfigurationCanLeaveOutIsChecked` fails if one doesn't.
- There is no option to turn the check off. Don't add one.
- A new top-level setting needs its default in the API's `defaults` response before the provider
  release. Without it, every tenant with any value for the setting fails its plan.
- The API reports no defaults for settings inside a nested attribute, so any value there fails the
  plan when the configuration leaves it out.
- Every value the check can list must be one the configuration can set. If validation rejects the
  value, the plan has to remove it instead. Example: a webhook URL stored under a provider other
  than `WEBHOOK`.
- A plan modifier that removes a value plans a known `null`, not unknown. The check skips a setting
  whose planned value is known and differs from state, because the plan already shows it.

## Create, Read, import and destroy

- Read fills state from the API for everything it returns, on import too. A value the
  configuration gets wrong then shows as a change, not as a new value.
- `ImportState` seeds a typed empty model, such as `emptyTenantSettingsModel`. An untyped nested
  object fails the Read that follows.
- Read removes the resource from state when it's gone. That means a 404, and also whatever marker
  the API uses for a soft delete.
- A guard goes in Create, Read and `ImportState`, not only Create. Example: `authsignal_rule`
  refuses a rule of a FLOW action in all three.
- Creating a resource over an object that already exists, such as an action, fails and says to
  import it. It doesn't overwrite.
- Creating a resource every tenant always has, such as tenant settings, reads the tenant's current
  settings and runs the unmanaged settings check. The first plan then fails before anything is
  written.
- A resource every tenant always has can't be destroyed. Fail the plan with `refuseDestroy`, which
  shows a `removed` block. Never write a Delete that does nothing, or one that resets some fields.

## Writes

- Send what the configuration sets, and send an explicit clear for what it stopped setting. Leaving
  a field out of the request keeps the tenant's value.
- Use `authsignal.SetNull` where the API accepts `null`. For credential members it doesn't, so they
  are cleared with an empty string or `false`, and Read treats those as unset.
- State can't show a block that has only secret members. Use a value the API derives from the secret
  as the sign that the tenant has it, and plan that value to `null` so the removal shows. Example:
  `fcm_project_id` and `fcm_client_email` for `fcm_credentials`.
- Write state from the API's response. A value planned as `null` has to come back absent, or the
  apply fails with "Provider produced inconsistent result after apply".

## Validation and error messages

- Fail at plan time for anything the API would reject at apply. Use `ValidateConfig` or an attribute
  validator.
- A validator, a plan modifier and `ModifyPlan` return early when a value they need is unknown. A
  value that comes from a variable or another resource is unknown at validation, so an error there
  fails a correct configuration.
- New validation must not block a change that used to work. Test the change from one valid
  configuration to another, such as a provider switch, not only the end state.
- `host` must be `https://`. Plain `http` is only for loopback addresses, which the stub tests use.

Every error follows these conventions:

- The summary is in Title Case, like "Settings Missing From the Configuration".
- The detail says what is wrong and what to do next. When the fix is a configuration change, show
  the HCL to paste.
- Offer a `removed` block or an `import` block, not `terraform state rm` or `terraform import`. A
  block shows in a plan and can be reviewed.
- Write a resource address as `<type>.<name>`, with `<name>` as the placeholder.
- Attach the attribute path, so Terraform points at the line.
- Never print a secret. A sensitive setting shows as `(sensitive value)`.

## Secrets and releases

- A secret never reaches state, an error message or a log line.
- Tests, examples and fixtures use made-up secrets only.
- Only ever test against a dev tenant. Never use a production tenant's key.
- Never put a key, a tenant ID, state or a variables file in a commit, a PR or a comment. Keep
  local test configurations outside the repository.
- Never create or push a tag. A `v*` tag publishes a signed release.
- Don't change `.github/workflows` unless asked. Actions stay pinned to a commit SHA.

## Tests

```bash
go vet ./...
go test ./...
TF_ACC=1 go test ./... -skip '^TestAcc'
go generate ./...
```

- Plain unit tests run with `go test ./...`.
- The plan tests run real Terraform against a local stub of the Management API. They need
  `TF_ACC=1` and a Terraform binary, 1.11 or later, but no tenant. CI runs them.
- The `TestAcc*` tests need a live tenant, and CI doesn't run them.
- A fix needs a test that fails on the old code. Undo the fix, watch the test fail, and restore it.
  A test that passes either way guards nothing.
- Cover the four questions under The bar with plan tests: import then plan, remove a setting and
  expect `-> null`, and `stub.assertNotCalled` for a plan that must not write.
- A stub test shows what the provider does with a given response. It doesn't show that the API
  gives that response. A change to what the provider sends also needs a check on a dev tenant.

## Docs

- `go generate ./...` writes `docs/` from the schema descriptions, `templates/` and `examples/`.
  Never edit `docs/` by hand. CI fails when it's out of date.
- A setting's description says what happens when the configuration leaves it out.
- Every example has to pass `terraform validate`.

## Versioning

- **Major**: anything that makes a passing plan fail, or changes what an apply does for an existing
  configuration. That covers removing or renaming an attribute, a new required attribute, stricter
  validation of a value that used to be accepted, a new or changed schema default, and clearing
  something the provider used to keep. A major release needs an upgrade guide in
  `templates/guides/`.
- **Minor**: new resources and new optional settings. Also deprecations: mark an attribute with
  `DeprecationMessage` in a minor release before a major release removes it.
- **Patch**: a fix that doesn't change what a passing plan does.

One exception, decided on 2026-10-29. A new optional setting on an existing resource ships in a
minor release, even though the unmanaged settings check then fails the plan for a tenant that has
a non-default value for it. The plan fails before anything is applied, it only happens after
`terraform init -upgrade`, and the error names the line to add. Such a release needs:

- the setting's default in the Management API's `defaults` first, and
- a CHANGELOG entry under ENHANCEMENTS, plus a NOTES entry that names the setting and says to add
  it to the configuration.

Every user-visible change gets a CHANGELOG entry under BREAKING CHANGES, NOTES, FEATURES,
ENHANCEMENTS or BUG FIXES.

`go.mod` has to name a released tag of `authsignal-management-go` before a PR merges, never an
unreleased build.

## Before a release

Test the release build with real Terraform against a dev tenant. The stub tests can't find a wrong
assumption about the API.

- Upgrade from state written by the previous release, with and without `-refresh=false`.
- Go back to the previous release with state written by the new one.
- For each changed resource: import with `-generate-config-out`, change a setting outside
  Terraform, remove a setting from the configuration, and switch provider.

## Known gaps

These don't meet the bar yet. Don't build on them, and remove an entry in the PR that fixes it.

- A rule added to a CLASSIC action outside Terraform doesn't show in a plan.
- After a rule is deleted, the API renumbers the action's other rules, so the next plan shows
  priority changes.
- A misspelt key inside a nested attribute is dropped by Terraform and plans as a removal.
- Some Computed attributes show `(known after apply)` on every update.

## Scope of changes

- Make the smallest diff that achieves the stated goal.
- Don't bundle refactors or extra hardening. List them as follow-ups and let a human decide.
- Only add a comment to explain a constraint the code can't show, and keep it to one line.

## Pull requests

- Open PRs as drafts. Prefix the title with the ticket reference, e.g. `[AUT-4282] Description`.
- The description says why the change exists and how the API's behaviour was checked. Don't restate
  the diff.
- Describe the API's behaviour, not its code. Never quote the API's code, file paths or internal
  route names.
- This repository is public. Don't name or link private repositories, customers or tenant IDs. Say
  "the Management API" and "the portal".
