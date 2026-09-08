---
name: spek-implement
description: Execute an approved Plan to implement the feature.
---

> **Version check first.** Before running any other command, run `spektacular version check`.
> - On `status: "match"`, continue with the skill and produce no version-related output.
> - On `"mismatch"` or `"missing"`, the installed Spektacular files are out of date: relay the response's `action` message to the user, ask them to re-run `spektacular init <agent>`, and wait for their decision before continuing.
> - Never modify or re-install any installed files yourself — refreshing the installation is always an explicit, user-initiated re-run of init.

> **STOP. Read this before running any command below.**
> A single successful CLI call — including the very first `implement new` — is **NOT** task completion. It is not a milestone to report back to the user. It is one step out of many in a workflow that you must keep driving, turn after turn, without stopping, until the CLI itself tells you the workflow is *finished*. If you find yourself about to say "successfully completed" or summarize results after calling `implement new` or `implement goto` even once, you are wrong — go back and read the `instruction` field you just received, do what it says, and call `goto` again.

# What this skill does

This skill drives a **multi-step interactive workflow** that executes an approved plan held in the plan store, producing working code, tests, and a changelog. The workflow is owned by the `spektacular` CLI, not by you — the CLI is the state machine and you are the executor, and the CLI (not the filesystem) is how you reach every plan document.

On each turn, the CLI returns JSON containing an `instruction` field. That instruction describes exactly one step (e.g. analyze, implement a phase, verify, update changelog, write the test plan, …). You must:

1. Read the `instruction` carefully.
2. Perform the step — this may mean reading the plan, spawning subagents, editing code, running tests, or writing to the changelog.
3. When the step is complete, run the `goto` command named at the bottom of the instruction to advance the state machine.
4. Read the next `instruction` from the new JSON response and repeat.

**This is a loop. Do not stop after the first step.** Keep looping — step → goto → next instruction → step — until a returned instruction tells you the workflow is *finished*. Only then should you report completion to the user.

**Concretely: do not stop after `implement new`.** That command only starts the workflow — it returns the *first* instruction, not a finished implementation. Seeing a clean JSON response with no `error` is not a signal to stop; it is the signal to keep going. Reporting success, summarizing "implementation initialized," or handing control back to the user at this point is the single most common way this skill is executed incorrectly — do not do it.

# Reading and writing plan files

The CLI owns the plan documents — `plan.md`, `context.md`, and `research.md`. **Never read or write them with the `Write`, `Edit`, or `Read` tools** — those bypass Spektacular and the configured plan directory. All plan document access goes through `spektacular plan file`:

- `spektacular plan file read <name>/<doc>.md` — read a plan document from the plan store.
- `spektacular plan file write <name>/<doc>.md --from <source-path>` — write a plan document into the plan store from a source file on disk. Stage the body under `.spektacular/tmp/` first, then `rm` the scratch file after a successful write.
- `spektacular plan file list` — list plans in the plan store.

This includes the edits the implement workflow makes to `plan.md` — ticking phase checkboxes and appending changelog entries. Read the document with `plan file read`, apply the change, and commit it with `plan file write`. Never edit a plan document in place with the `Edit` tool. Path arguments are plan-directory-relative document paths (e.g. `my-feature/plan.md`).

# How to start

> **Cross-repo implementation.** When the plan attributes work to registered member repos, carry each part of the work out in its attributed repo's code (`spektacular repo list` reports where it lives as `root`), and follow the workflow's changelog instructions to write the central record plus one derived entry per affected repo via `spektacular changelog file write ... --repo <name>`.

Ask the user which plan to implement before proceeding. To enumerate the available plans, run `spektacular plan file list` — the CLI's list is the source of truth for what counts as a plan. **Do not** use `ls`, `find`, or the `Read` tool against `.spektacular/plans/` to discover plans; those bypass Spektacular's configured plan directory and may show entries the CLI does not consider valid. You don't need to look for an in-progress workflow yourself — the CLI detects and reports one for you (see below).

The plan must already exist in the plan store — confirm with `spektacular plan file list`. If it does not, stop and tell the user to run `spektacular plan` first.

Start the implement workflow by running:

```
spektacular implement new --data '{"name": "<plan_name>"}'
```

**If a workflow was interrupted and is still in progress**, this command does not start a fresh one. Instead it returns a *resume report* — a JSON object with `"resumable": true` plus the in-progress workflow's `kind`, `name`, and `current_step`, and an `instruction` field — and changes nothing on disk. When you get a resume report:

**First check the report's `kind`.** If it is **not** `implement`, a *different* workflow (a spec or plan run) is in progress — you cannot resume it from the implement skill, and the CLI will refuse to. Do **not** run an `implement goto`. Instead follow the report's `instruction`: tell the user a `<kind>` workflow is in progress and let them choose — continue it with that workflow's skill (`spektacular <kind> goto`), or discard it and start the implement run with `spektacular implement new --force`. Only proceed with the steps below when the report's `kind` is `implement`.

1. Ask the user whether to **resume** the in-progress implement run or **start a new one**. (The report's `instruction` field restates both options.)
2. **To resume**, first read `.spektacular/context.md` — the git-tracked working-context file the previous session left behind — to recover its learnings and the answers you gave to the user's questions, then run the resume command using the report's `current_step`:

   ```
   spektacular implement goto --data '{"step":"<current_step>"}'
   ```
3. **To start fresh** (discarding the in-progress workflow — it remains recoverable via git), re-run with `--force`:

   ```
   spektacular implement new --force --data '{"name": "<plan_name>"}'
   ```

Otherwise the command returns the first `instruction` and a fresh workflow has started. From that point on, follow the loop above: do what the instruction says, then call `spektacular implement goto --data '{"step":"<next_step>"}'` to get the next one. Do not invent step names — every instruction tells you the exact `goto` command to run next.
