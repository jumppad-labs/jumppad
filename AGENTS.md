## Where the Code Lives — Read This First

> Managed by `spektacular init` — edit `templates/agents/repo-sources.md`
> in the Spektacular source, not this section in place. Hand edits will not
> survive the next init.

**STOP. The directory you are running in is not necessarily the code.**

This is a Spektacular project. The code it describes may live anywhere on
disk — a sibling directory, somewhere else entirely on the filesystem, or a
clone this project manages. A project directory holding only
`.spektacular/`, `knowledge/`, and `repos/` is normal, and is **not** an
empty or broken repository: the code is elsewhere, and the project knows
where.

**Before any code-touching work — searching, reading, grepping, editing,
running tests, verifying, or answering a question about how something
works — run:**

```
spektacular repo list
```

Every registered repo comes back with a `root`: the absolute path where
that repo's code actually lives. Work in that `root`. Cite files by paths
under it. Run tests from it.

These rules are not negotiable:

- **Never assume the directory you started in holds a repo's code.** Check
  with `spektacular repo list` first, in every session, before you open the
  first file.
- **Never guess a path** from the project's name, a repo's name, or a
  directory that looks plausible. The registry is the only authority on
  where code lives.
- **Pass the relevant repo's `root` explicitly to every sub-agent you
  launch.** A sub-agent inherits your working directory, not your knowledge
  of where the code is.
- **If a repo's `root` is not present on disk, stop and tell the user.**
  Report the path the registry gave and that it is missing. Do not
  substitute a directory that looks close, and do not silently carry on in
  the project directory instead.

This rule outranks your default assumptions about working directories, and
it binds everywhere in this project — inside spec, plan and implement
workflows, and equally in ad-hoc questions, unrelated skills, and general
exploration. It is not limited to workflow steps.

## Memory & Context

> Managed by `spektacular init` — edit `templates/agents/memory-context.md`
> in the Spektacular source, not this section in place. Hand edits will not
> survive the next init.

In this repository, do not persist anything to your per-user, per-machine
memory store. When you would normally write to it — a learning, convention,
gotcha, project fact, user preference, or anything else worth remembering
between sessions — route the write through the `spek-knowledge` skill
instead. The skill handles tier and store selection, search-before-write, and
propose-then-confirm.

Outside this repository, continue using your per-user memory store as normal.

## Knowledge-Worthy Discovery Recognition

> Managed by `spektacular init` — edit `templates/agents/knowledge-trigger.md`
> in the Spektacular source, not this section in place. Hand edits will not
> survive the next init.

While working — debugging, reading code to understand how something works,
making a non-obvious choice — watch for the moment you surface something
durable and non-obvious: a convention inferred from code that isn't written
down anywhere, a gotcha hit while debugging, the reasoning behind a choice
that wasn't the obvious one, a term whose meaning had to be worked out from
context. Recognizing this moment is your job, not the user's — don't wait to
be asked.

When you recognize the moment, offer — never write to the knowledge base
unprompted. Something like: "this looks like an undocumented convention —
want me to save it via `spek-knowledge`?" Say briefly what you'd capture and
why it's worth keeping. Wait for the user's decision before doing anything
else.

The user's response falls into one of three outcomes:

- **Accept** — invoke the `spek-knowledge` skill to write the entry. The
  skill's own propose-then-confirm flow handles tier and store selection and the
  actual write from there.
- **Defer** ("not now", "later", "remind me at the end") — do not invoke the
  skill. Continue the task normally, and treat this as temporary: if the
  work keeps surfacing related material, you may raise the offer again later
  in the same conversation.
- **Decline** ("no", "not worth saving") — do not invoke the skill, and do
  not raise the offer again for this discovery for the remainder of the
  conversation. A decline is final for that discovery, not a "not now."

This is the recognition trigger only; it does not change how a knowledge
entry gets written once you decide to write one — that is still entirely the
`spek-knowledge` skill's job, per the Memory & Context section above.

## Spec-Worthy Discussion Recognition

> Managed by `spektacular init` — edit `templates/agents/spec-trigger.md`
> in the Spektacular source, not this section in place. Hand edits will not
> survive the next init.

During an open-ended discussion (diagnostics, brainstorming, exploratory back-and-forth), watch for the moment it produces something substantial enough to be worth capturing as a specification — multiple requirements mentioned, a scoped decision reached, or a feature described in enough detail that it could be built from. Don't wait to be asked; recognizing this moment and offering is your job, not the user's.

This check is not limited to live back-and-forth discussion. It applies just as much the moment you finish reading a fully-formed request — a GitHub issue, a linked ticket, a pasted design doc — that already describes a feature in spec-worthy detail. Reading a well-specified issue and going straight to implementation is the same miss as skipping the offer mid-conversation: evaluate the request against the criteria above *before* starting any implementation work, not only when the detail accumulates turn-by-turn in front of you.

Before deciding whether a discussion has crossed that line, read `spec_trigger_threshold` from `.spektacular/config.yaml` — check it at the moment you are deciding, not once at the start of the session, since the user may change it mid-conversation and expects that to take effect immediately. Treat a missing or absent value as `"moderate"`. Use the configured value to calibrate how readily you offer:

- `"strict"` — only offer for substantial, multi-requirement features. Small fixes and minor tweaks should not trigger an offer.
- `"moderate"` — offer once a discussion reaches a clear, scoped decision or a feature description with more than one requirement. The default balance.
- `"lenient"` — offer readily, including for small bug fixes or narrowly scoped changes, whenever there's a decision worth recording.

When you recognize the moment, always offer — never start the spec workflow unilaterally. Propose capturing the discussion as a spec, briefly say why (e.g. "this sounds like it's grown into a few concrete requirements — want me to capture it as a spec?"), and wait for the user's decision before doing anything else.

The user's response falls into one of three outcomes:

- **Accept** — proceed to carry-forward below.
- **Defer** ("not yet", "still investigating", "let me think") — do not start the spec workflow. Continue the conversation normally, and treat this as temporary: if the discussion keeps developing, you may raise the offer again later in the same conversation.
- **Decline** ("no", "I don't want a spec for this") — do not start the spec workflow, and do not raise the offer again for this discussion topic for the remainder of the conversation. A decline is final for that topic, not a "not now."

If the user accepts, start the spec workflow:

```
spektacular spec new --data '{"name":"..."}'
```

Then drive its existing steps — the "ask the user..." prompts in `templates/steps/spec/*.md` are unchanged and still apply — but answer each one from the conversation you already had instead of asking cold. For every step where the discussion already established an answer (e.g. the overview step's "describe this feature in 2-3 sentences"), propose a draft based on what was said and ask the user to confirm or refine it, rather than posing the question as if from scratch. Only ask a step's question directly when the conversation genuinely didn't cover it. The user's confirmation or correction is still the final word — never silently record your own draft as accepted without it.

This behavior is scoped to the current, single conversation: it does not persist across sessions, and it does not apply outside a Spektacular-initialized repository.

## Presenting Drafts and Confirmations

> Managed by `spektacular init` — edit `templates/agents/draft-presentation.md`
> in the Spektacular source, not this section in place. Hand edits will not
> survive the next init.

When you have drafted substantial content for the user to review — an architecture write-up, a set of options, a written section, a summary, a plan or spec excerpt — always present that draft as normal, readable chat text first, in full. Never embed the draft itself inside a structured yes/no or multiple-choice dialog element: that kind of UI truncates or compresses long text, making it hard for the user to actually read what you're proposing.

Once the draft has been shown in full as plain text, you may then ask a short, direct confirmation question — e.g. "does this look right, or should anything change?" — using a structured yes/no or multiple-choice element if one is available. That confirmation step comes strictly after the content has already been presented as text, and is never a substitute for showing it.

This rule applies across every workflow step that follows a draft-then-confirm pattern — spec, plan, and implement workflows alike — not just one. If a specific step's own instructions don't repeat this explicitly, this standing rule still applies.

## Historical Artifacts: Specs and Plans as Archaeology

> Managed by `spektacular init` — edit `templates/agents/historical-artifacts.md`
> in the Spektacular source, not this section in place. Hand edits will not
> survive the next init.

In this repository, treat every file under `.spektacular/specs/` and
`.spektacular/plans/` as a historical, archaeological record. Each one
describes the intent behind a past change — *why* something was proposed,
what was in scope at that moment, and how the author framed the problem.
None of them are a description of what the codebase does today. Intent
recorded in a spec or plan may have been reshaped during implementation,
descoped, or abandoned entirely; only the shipped code, its tests, and its
configuration authoritatively describe current behavior.

Because of that, when you are exploring the codebase, summarizing a
feature, tracing how something works, or answering any question about
current-state behavior, do not read files under `.spektacular/specs/` or
`.spektacular/plans/` — through the `Read` tool, through `spektacular spec
file read`, through `spektacular plan file read`, or through any other
channel. Ground your answer in source files, tests, and configuration
instead, and cite paths under those directories rather than under the
spec or plan stores.

You may read a historical spec or plan only when the user is genuinely
investigating past intent — questions like "why was X built this way?",
"what was the original plan for Y?", or "which spec introduced Z?". In
that case, read the relevant document, and cite it explicitly as
historical context for a past decision rather than as a description of
current behavior. Archaeology is the only allowed reason to open these
files outside an active workflow.

There is one further exception: while a spec, plan, or implement
workflow is actively running, the workflow that owns its artifact may
read and update that artifact freely. That is what the workflow is for,
and it uses the dedicated CLI (`spektacular spec file read/write`,
`spektacular plan file read/write`) to do so. Once the workflow closes —
or for any agent that is not the workflow currently driving the
artifact — the artifact is historical again and subject to the same
rules as every other spec or plan on disk.

This rule applies everywhere you operate in the repository, not only
inside spec, plan, or implement workflow steps. It binds ad-hoc
questions, unrelated skills, and general exploration alike. Users
should not have to restate it in each session.

More broadly, the rest of `.spektacular/` — knowledge entries,
`context.md`, changelog records — is generated output *about* the
codebase, not the codebase itself. A broad grep or file scan run to
understand current-state behavior should treat all of `.spektacular/`
as out of scope, the same way it treats `.spektacular/specs/` and
`.spektacular/plans/` above, unless the task explicitly concerns specs,
plans, or knowledge.
