---
name: spek-plan
description: Create a new Plan from an approved Specification.
---

> **Version check first.** Before running any other command, run `spektacular version check`.
> - On `status: "match"`, continue with the skill and produce no version-related output.
> - On `"mismatch"` or `"missing"`, the installed Spektacular files are out of date: relay the response's `action` message to the user, ask them to re-run `spektacular init <agent>`, and wait for their decision before continuing.
> - Never modify or re-install any installed files yourself — refreshing the installation is always an explicit, user-initiated re-run of init.

> **STOP. Read this before running any command below.**
> A single successful CLI call — including the very first `plan new` — is **NOT** task completion. It is not a milestone to report back to the user. It is one step out of many in a workflow that you must keep driving, turn after turn, without stopping, until the CLI itself tells you the workflow is *finished*. If you find yourself about to say "successfully completed" or summarize results after calling `plan new` or `plan goto` even once, you are wrong — go back and read the `instruction` field you just received, do what it says, and call `goto` again.

# What this skill does

This skill drives a **multi-step interactive workflow** that produces a complete implementation plan — the assembled `plan.md`, `context.md`, and `research.md` documents committed to the plan store — from an existing spec. The workflow is owned by the `spektacular` CLI, not by you — the CLI is the state machine and you are the executor, and the CLI (not the filesystem) is how you reach every plan document.

On each turn, the CLI returns JSON containing an `instruction` field. That instruction describes exactly one step (e.g. discovery, data structures, phases, testing approach, walkthrough, …). You must:

1. Read the `instruction` carefully.
2. Perform the step — this may mean researching the codebase, spawning subagents, interviewing the user, or committing a plan document to the store.
3. When the step is complete, run the `goto` command named at the bottom of the instruction to advance the state machine.
4. Read the next `instruction` from the new JSON response and repeat.

**This is a loop. Do not stop after the first step.** Keep looping — step → goto → next instruction → step — until a returned instruction tells you the workflow is *finished*. Only then should you report completion to the user.

**Concretely: do not stop after `plan new`.** That command only starts the workflow — it returns the *first* instruction (the `overview` step), not a finished plan. Seeing a clean JSON response with no `error` is not a signal to stop; it is the signal to keep going. Reporting success, summarizing "plan initialized," or handing control back to the user at this point is the single most common way this skill is executed incorrectly — do not do it.

**The workflow ends with a mandatory `walkthrough` review.** After the three documents are committed to the store, the CLI renders the `walkthrough` step: walk the user through the committed plan section by section, apply any requested changes immediately through `spektacular plan file write`, and only advance to `finished` once the user gives an explicit affirmative answer to a direct closing question. Committed documents are **not** completion — the workflow is finished, and the plan approved, only after the user signs off during the walkthrough and the `finished` step has run.

# Reading and writing plan files

The CLI owns the plan documents — `plan.md`, `context.md`, and `research.md`. **Never read or write them with the `Write`, `Edit`, or `Read` tools** — those bypass Spektacular and the configured plan directory. All plan document access goes through `spektacular plan file`:

- `spektacular plan file read <name>/<doc>.md` — read a plan document from the plan store.
- `spektacular plan file write <name>/<doc>.md --from <source-path>` — write a plan document into the plan store from a source file on disk. Stage the body under `.spektacular/tmp/` first, then `rm` the scratch file after a successful write.
- `spektacular plan file list` — list plans in the plan store.

Path arguments are plan-directory-relative document paths (e.g. `my-feature/plan.md`); `plan file` resolves them against the configured plan directory itself.

# Working files vs. the store documents

The drafting steps run without stopping for section approval — draft each section, save it, and advance; only a genuinely blocking question (no reasonable default, or information only the user holds) interrupts the user before the walkthrough.

While you gather each section, write that section's drafted content directly to its own git-tracked working file under `.spektacular/work/<plan_name>/<section>.md` using your own `Write` tool (the phases step writes two: `phases_plan.md` and `phases_context.md`; every drafting step also appends its judgement calls to a shared `assumptions.md` in the same directory). These working files are **not** store documents — writing them directly with `Write` is correct and expected, and is the one deliberate exception to the "never use `Write`/`Edit`" rule above. That rule protects only the **final assembled** `plan.md`, `context.md`, and `research.md`, which are written solely through `spektacular plan file write`. The per-section working files are scratch-but-durable: the assemble step reads them back to build the three documents (staged to `.spektacular/tmp/`), the verification step checks the staged documents, the write steps commit them, and then the working directory is removed once all three store writes succeed.

The working sidecar `.spektacular/context.md` (at the repo's `.spektacular/` root — not the plan's own `context.md` document) has a narrower role: it holds only your cross-cutting learnings and the answers the user gave to your questions — never a copy of section content (that lives in the per-section working files). On resume, read back **both** the section working files in `.spektacular/work/<plan_name>/` (including the `assumptions.md` judgement-call log) and `.spektacular/context.md`, so you continue from the interrupted step without re-asking for sections already completed or re-deciding calls already recorded.

# How to start

> **Cross-repo planning.** A project may register multiple member repos (see `spektacular repo list`). The workflow's discovery and architecture instructions send you to `spektacular repo list` and direct you to attribute every requirement to the repo (and files) it belongs to — research across all registered repos, in the `root` reported for each, and record the attribution in the plan's context document.

Ask the user which spec to plan against before proceeding. To enumerate the available specs, run `spektacular spec file list` — the CLI's list is the source of truth for what counts as a spec. **Do not** use `ls`, `find`, or the `Read` tool against `.spektacular/specs/` to discover specs; those bypass Spektacular's configured spec directory and may show entries the CLI does not consider valid. You don't need to look for an in-progress workflow yourself — the CLI detects and reports one for you (see below).

Start the plan workflow by running:

```
spektacular plan new --data '{"name": "<spec_name>"}'
```

**If a workflow was interrupted and is still in progress**, this command does not start a fresh one. Instead it returns a *resume report* — a JSON object with `"resumable": true` plus the in-progress workflow's `kind`, `name`, and `current_step`, and an `instruction` field — and changes nothing on disk. When you get a resume report:

**First check the report's `kind`.** If it is **not** `plan`, a *different* workflow (a spec or implement run) is in progress — you cannot resume it from the plan skill, and the CLI will refuse to. Do **not** run a `plan goto`. Instead follow the report's `instruction`: tell the user a `<kind>` workflow is in progress and let them choose — continue it with that workflow's skill (`spektacular <kind> goto`), or discard it and start the plan with `spektacular plan new --force`. Only proceed with the steps below when the report's `kind` is `plan`.

1. Ask the user whether to **resume** the in-progress plan or **start a new one**. (The report's `instruction` field restates both options.)
2. **To resume**, first read back the previous session's work with your own file tools: the per-section working files under `.spektacular/work/<name>/` (sections already completed) **and** `.spektacular/context.md` (learnings + the user's answers). If the report's `current_step` is `walkthrough`, the per-section working files have already been removed — read the committed documents back with `spektacular plan file read <name>/<doc>.md` instead, then continue the interrupted review from there. Then run the resume command using the report's `current_step`:

   ```
   spektacular plan goto --data '{"step":"<current_step>"}'
   ```
3. **To start fresh** (discarding the in-progress workflow — it remains recoverable via git), re-run with `--force`:

   ```
   spektacular plan new --force --data '{"name": "<spec_name>"}'
   ```

Otherwise the command returns the first `instruction` and a fresh workflow has started. From that point on, follow the loop above: do what the instruction says, then call `spektacular plan goto --data '{"step":"<next_step>"}'` to get the next one. Do not invent step names — every instruction tells you the exact `goto` command to run next.
