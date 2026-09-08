---
name: spek-manage-repos
description: Add a repo to the current Spektacular project through a guided conversation, inspect the registry, and repair a repo's footprint.
---

> **Version check first.** Before running any other command, run `spektacular version check`.
> - On `status: "match"`, continue with the skill and produce no version-related output.
> - On `"mismatch"` or `"missing"`, the installed Spektacular files are out of date: relay the response's `action` message to the user, ask them to re-run `spektacular init <agent>`, and wait for their decision before continuing.
> - Never modify or re-install any installed files yourself. Refreshing the installation is always an explicit, user-initiated re-run of init.

> **STOP. Read this before running any command below.**
> A single successful CLI call, including the very first `repo new`, is **NOT** task completion. It is not a milestone to report back to the user. It is one step out of many in a workflow that you must keep driving, turn after turn, without stopping, until the CLI itself tells you the workflow is *finished*. If you find yourself about to say "successfully completed" or summarize results after calling `repo new` or `repo goto` even once, you are wrong. Go back and read the `instruction` field you just received, do what it says, and call `goto` again.

# What this skill does

This skill covers the `spektacular repo` surface: adding a repo to the current project, inspecting what is already registered, and repairing a repo whose footprint is missing or broken.

Adding a repo is a **multi-step interactive workflow** owned by the CLI, not a playbook you improvise from. The CLI is the state machine and you are the executor. On each turn it returns JSON containing an `instruction` field describing exactly one step. You must:

1. Read the `instruction` carefully.
2. Perform the step. Usually that means asking the user exactly one thing, or recording something without asking.
3. When the step is complete, run the `goto` command named at the bottom of the instruction, carrying the answer you just agreed.
4. Read the next `instruction` from the new JSON response and repeat.

**This is a loop. Do not stop after the first step.** Keep looping, step then goto then next instruction then step, until a returned instruction tells you the workflow is *finished*. Only then report completion to the user.

**Concretely: do not stop after `repo new`.** That command only starts the workflow and returns the *first* instruction, not a registered repo. Seeing a clean JSON response with no `error` is not a signal to stop; it is the signal to keep going.

The instructions themselves tell you what to ask and what never to say. Follow them as written rather than substituting your own account of how an add works: the wording is the feature.

Inspecting the registry and repairing a footprint are not workflows. They are single commands, covered further down.

# When to invoke

- "Add a repo." / "Register the docs repo." / "Add this project to Spektacular."
- "What repos are in this project?" / "Where does the API repo's code live?"
- Any `repo_footprint` or `repo_footprint_missing` error returned by another command.
- Before attributing work to a repo, when you need each repo's resolved `root`.

# Concepts

A **project** is a collection of repos with central spec, plan, and changelog storage. A **repo** joins a project by being registered in the project's `config.yaml`. Each registered entry carries a slug-safe `name` (plans and changelog attribution reference repos by name), a `location`, and optional `dependencies`.

A repo has two locations, and keeping them apart is the whole trick:

- The **footprint** is the folder holding the repo's `repo.yaml`, alongside its knowledge and changelog. This is what the registry's `location` records. (The older `local` key still works and means the same thing.)
- The **source** is where the repo's *code* lives. It is declared inside that `repo.yaml` as a provider block, the same shape every other section uses: `provider: file` with a `config.location` path, or `provider: git` with a `config.location` git address that is cloned on registration.

A repo is **colocated** when its footprint sits inside its code, in a `.spektacular/` folder whose `repo.yaml` declares a file source of `..`. This is what `repo add` scaffolds. A repo is **separate** when its footprint lives in a folder of its own and its `source` points at code elsewhere, so the code repository receives only code changes.

Descriptive metadata (`description`, `role`, `tags`) lives in `repo.yaml`, never in the project config. A repo's footprint carries no pointer back to any project, so one repo can belong to many projects.

# Adding a repo

Start the guided add by running:

```
spektacular repo new
```

If the user already named the repo, pass the folder its code lives in and the flow will not ask for it again:

```
spektacular repo new --data '{"location":"<the folder the repo's code is in>"}'
```

From there, follow the loop above: do what the instruction says, then run the `goto` it names to get the next one. Do not invent step names. Every instruction ends with the exact command to run next.

**If an add was interrupted and is still in progress**, `repo new` does not start a fresh one. It returns a *resume report*: a JSON object with `"resumable": true` plus the in-progress workflow's `kind`, `name`, and `current_step`, and an `instruction` field. Nothing on disk changes. When you get one:

**First check the report's `kind`.** If it is not `repo`, a different workflow is in progress and you cannot resume it from here. Follow the report's `instruction`: tell the user which workflow is in progress and let them choose to continue it with that workflow's skill, or to discard it. Only proceed below when the report's `kind` is `repo`.

1. Ask the user whether to **resume** the in-progress add or **start a new one**. The report's `instruction` restates both options.
2. **To resume**, first read `.spektacular/context.md` with your own file tools, for the cross-cutting learnings and the answers the user gave you. Unlike a spec or a plan, an add has no per-section working files to read back: every answer already agreed travels inside the workflow itself and comes back with it. Then run the resume command using the report's `current_step`:

   ```
   spektacular repo goto --data '{"step":"<current_step>"}'
   ```

3. **To start fresh**, discarding the in-progress add, re-run with `--force`:

   ```
   spektacular repo new --force
   ```

Nothing is written to the user's repo or to the project until the confirmation step has been passed, so an add abandoned partway through leaves no trace to clean up.

**The single-command form is still there** for a caller that already knows every detail and wants no conversation:

```
spektacular repo add --data '{"name":"<name>","location":"<folder>","description":"<description>","role":"<role>","tags":["<tag>"]}'
```

Prefer the guided add when a person is involved. Use the direct form when scripting, or when every value is already known and settled.

# Rules that apply to every add

- Registration is idempotent. Re-adding the same entry changes nothing; re-adding with different metadata updates it in place; adding a repo another project already initialized registers it here without disturbing its existing footprint. Re-adding without `source` leaves the stored source alone.
- A git source is cloned on registration, so the first add of a git-source repo takes as long as a clone.
- The location is resolved as part of the add, so a footprint that cannot be created or a clone that fails surfaces immediately rather than later.
- A payload carrying `address` is rejected with the corrected payload. A repo's git origin is its `source`.

# Inspecting the registry

```
spektacular repo list
```

Each entry reports:

- `name`, and the resolved `location` of its footprint;
- the resolved source of its code as `root`, which is the file source, the clone of a git source, or the location itself when the repo declares no source;
- the `provider` the repo declares for its source (`file`, `git`, or absent when it declares none);
- `description`, `role`, `tags`, and `dependencies`;
- `materialized`, true when the root is a project-managed clone;
- `stale_note` when a clone has fallen behind its remote;
- `metadata_note` when the repo has no descriptive metadata set.

Listing does not clone: a repo whose git source has not been materialized yet reports an empty `root` rather than triggering a clone. A registered repo whose footprint is missing is reported as a `repo_footprint_missing` error naming the path that was looked at, not listed with an empty root as if that were fine.

# Materialization and staleness

- A repo with no `source`, or with a file source, resolves to that directory. Git is never involved.
- A repo whose `source` is a git location is cloned into `.spektacular/repos/<name>/` inside the project. Project init gitignores that folder, so clones never enter the project's history.
- Spektacular never fetches or pulls on its own. A stale clone produces a `stale_note` warning and nothing more. If the user wants it updated they update it themselves, for example `git -C .spektacular/repos/<name> pull`. Always confirm with the user before suggesting a command that changes a clone.

# Footprint repair

Touching a registered repo whose `repo.yaml` is missing or invalid produces a structured error offering repair, never a silent failure: `repo_footprint_missing` from `repo list`, and `repo_footprint` from the knowledge and store-file commands. To repair, either re-run `repo add` with the repo's name and its code location (the registration is preserved; only the footprint is recreated), or re-run `spektacular init <agent>`, which cascades over every registered repo and repairs their footprints.

If the reported path is wrong rather than missing, the fix is the registry, not the footprint: correct that repo's `location` in the project's `config.yaml`.

# Removal

Removal is deliberately a manual edit: delete the repo's entry from the `repos` list in the project's `config.yaml`. No command is provided, and nothing in the repo itself needs cleaning up, since its footprint carries no project pointer. Confirm with the user before editing their config.
