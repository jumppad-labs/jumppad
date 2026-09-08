---
name: spek-knowledge
description: Search, contribute to, or update the project's knowledge base.
---

> **Version check first.** Before running any other command, run `spektacular version check`.
> - On `status: "match"`, continue with the skill and produce no version-related output.
> - On `"mismatch"` or `"missing"`, the installed Spektacular files are out of date: relay the response's `action` message to the user, ask them to re-run `spektacular init <agent>`, and wait for their decision before continuing.
> - Never modify or re-install any installed files yourself — refreshing the installation is always an explicit, user-initiated re-run of init.

# What this skill does

This skill orchestrates the existing `spektacular knowledge` CRUD surface for ad-hoc read, contribute, and update operations on the project's knowledge store, without starting a spec/plan/implement flow. Unlike `spek-new`, `spek-plan`, and `spek-implement`, it does not drive an interactive CLI state machine — it is a static playbook. The agent recognises the user's natural-language intent, picks one of three branches (lookup / contribute / update), and calls the matching `spektacular knowledge` command directly.

# When to invoke

Invoke this skill any time the user references the knowledge base, an entry, a convention worth remembering, or asks a question that the knowledge store might already answer. Typical natural-language triggers include:

- "What do we know about X?"
- "Search the knowledge base for Y."
- "Remember that Z is the case." / "Add a note about Z."
- "Update what we have on W."
- "Recall the convention for V."

One skill handles all three intents. Discriminate by what the user actually said — do not ask the user to pick a slash command per intent.

# Intent: lookup

Triggered when the user wants to read or search existing entries. A lookup does **not** dump the raw hit list — it returns a single consolidated, source-cited answer with duplicates removed and every contributing store cited. The flow has a deterministic stage (exact de-dup) and a judgement stage (consolidation), kept strictly separate.

1. **Search.** Run `spektacular knowledge search <query>` with a concise query derived from the user's question. Narrow it with `--tier <project|repo|all>` and a repeatable `--filter <store>` when the question is about particular repos; omitting both covers every configured store. The output is a ranked list of results — one per matching document, strongest match first — each carrying its `tier` and `name` (the store it came from), `path`, `title`, `score`, `category` (the kind of knowledge: e.g. `gotchas`, `architecture`, `learnings`), `checksum` (a content hash), and up to three `excerpts`. A document matches when every query word occurs somewhere in it, in any order. If there are no hits, say so plainly and stop — do not fall back to a write unless the user explicitly asks to add a new entry.

2. **Exact de-dup (deterministic — no judgement).** Group the hits by their `checksum`. Hits sharing a checksum are byte-identical copies of the same entry held in more than one place; collapse each such group to a **single candidate**, unioning the `tier`/`name`/`path` citations of every copy in the group. This is pure equality — never merge two entries whose checksums differ at this stage, however similar they look. The result is a list of unique candidates, each with one or more source citations.

3. **Consolidate (judgement — delegated to a sub-agent).** Hand the unique candidates to a consolidation **sub-agent** so the raw bodies never crowd the main context. The sub-agent's contract:
   - **Input:** the user's question and the list of unique candidates (each with its `tier`, store `name`, `path`, and `category`).
   - **Task:** read each candidate's full body with `spektacular knowledge read --data '{"tier":"<tier>","name":"<name>","path":"<path>"}'`, then classify the *relationship* between candidates and combine them:
     - **Equivalent** (same point, different words) → merge into one point, citing every source.
     - **Refinement** (one is a more specific case of another) → keep both and say which is the narrower case. **There is no precedence between stores**: a repo's own store answers what is true of that repo's code, and the project's shared stores answer what spans repos. Neither overrides the other, so never silently drop one because of where it lives.
     - **Genuine contradiction** (sources actually disagree) → **surface it explicitly** as a conflict naming both stores; never silently drop or average it.
     - **Distinct** (unrelated points) → keep both.
   - **Output (returned to the main agent):** a single consolidated answer composed of merged points, **each citing the tier, store name and path** it was drawn from, with any contradictions presented as surfaced conflicts. The raw per-source candidate list is **not** the output.

4. **Present** the sub-agent's consolidated answer to the user, keeping every **citation (tier, store name and path) visible** so the user can see which configured store each point came from. Never present the raw hit list as the result.

**If the executing agent cannot spawn a sub-agent**, run the exact same consolidation inline in the main context instead: read the unique candidates' bodies, apply the identical relationship-classification rules, and present the same single cited answer. The output is identical; only the context isolation is weaker. Do not block on the absence of sub-agent orchestration.

# Intent: contribute

Triggered when the user wants to record something new.

1. Run `spektacular knowledge sources` to enumerate the configured stores by `tier` and `name`. That listing is the authoritative set of writable destinations and of valid `--filter` names: choose only from the names it returns, never a name you inferred. Run `spektacular knowledge categories` to load the category definitions — each category's purpose, boundary, retrieval tier, and expected entry shape.
2. **Route the entry to a category.** From the definitions, pick the category whose **Purpose** matches the entry and whose **Boundary** does not push it elsewhere — e.g. a standing rule is a `convention`, a defined term is a `glossary` entry, the reasoning behind a choice is a `decision`, an empirical finding is a `learning`, a structural fact is `architecture`, a sharp edge is a `gotcha`. Honour the **entry shape**: the `glossary` is for a term and a short gloss only — steer over-long or multi-paragraph content to a more fitting category (architecture, learnings, decisions) rather than letting it bloat the always-applied glossary. The entry's path is then `<category>/<slug>.md`, a slug-style filename under the chosen category.
3. Decide (or ask the user) which **store** to write to — a `tier` and a `name` from the enumeration in step 1 — and the entry **body**. Knowledge about one repo's own code belongs in that repo's store, under the name the project registered it by; knowledge that spans repos belongs in one of the project's shared stores. The path comes from the category routing in step 2.
4. Stage the body on disk under `.spektacular/tmp/<slug>.md` using the `Write` tool. Do not pipe the body via stdin; the only supported invocation is `--file <staged>`.
5. **Show the user the destination and the body before writing.** The destination must state the **tier**, the **store name**, and the **path** — approval is for where the entry lands, not just what it is called. Wait for **explicit confirmation** ("yes", "go ahead", or equivalent). If the user asks for changes, revise the staged body and re-show — never write on an implicit signal.
6. Only after explicit confirmation, run:
   ```
   spektacular knowledge write --data '{"tier":"<tier>","name":"<name>","path":"<category>/<slug>.md"}' --file .spektacular/tmp/<slug>.md
   ```
   A write that leaves out the tier or the store name is refused, and the refusal lists the names available in that tier.
7. Remove the scratch file after a successful write: `rm .spektacular/tmp/<slug>.md`.

# Intent: update

Triggered when the user wants to revise an existing entry.

1. Identify the target entry. Run `spektacular knowledge search <query>` (or read the user-supplied path directly) to locate it. A hit carries its own `tier`, `name` and `path`, which is everything a read needs, so no further disambiguation is required. Confirm the store and path with the user if there is any ambiguity.
2. Read the current body with `spektacular knowledge read --data '{"tier":"<tier>","name":"<name>","path":"<path>"}'`.
3. Apply the user's revision intent to produce new content. Stage the revised body under `.spektacular/tmp/<slug>.md` using the `Write` tool.
4. **Show the user the tier, store name, path, and proposed new body (or a diff against the current body) before writing.** Wait for **explicit confirmation**.
5. Only after explicit confirmation, run:
   ```
   spektacular knowledge write --data '{"tier":"<tier>","name":"<name>","path":"<path>"}' --file .spektacular/tmp/<slug>.md
   ```
   The tier, name and path must all match the original — that is what makes this an update rather than a new entry somewhere else.
6. Remove the scratch file after a successful write: `rm .spektacular/tmp/<slug>.md`.

# Decline handling

If the user declines, asks for changes, or expresses uncertainty at any propose-then-confirm checkpoint, **do not invoke `spektacular knowledge write`**. Either loop back to refine the proposal — adjust the tier, store name, path, or body and re-show — or stop and leave the knowledge store untouched. Removing the staged scratch file at `.spektacular/tmp/<slug>.md` is fine either way; a half-finished proposal should not linger on disk.

The propose-then-confirm contract is enforced by this prose, not by a CLI guard. Treat it as load-bearing: a write without explicit user approval is a bug in the skill's execution, not an acceptable shortcut.
