---
name: implementation-planner
description: Create detailed implementation plans through an interactive process with research, code snippets, and structured deliverables. Use this skill when planning significant features, refactoring tasks, or complex implementations that require thorough analysis and structured documentation. The skill guides through context gathering, research, design decisions, and generates comprehensive plans with test strategies and success criteria.
---

# Implementation Planner

## Overview

Create detailed implementation plans through an interactive, iterative process. Be skeptical, thorough, and work collaboratively with the user to produce high-quality technical specifications with proper separation of working notes from deliverables.

**Language-Agnostic Approach:** This skill is language-agnostic and delegates to language-specific guidelines skills (e.g., `go-dev-guidelines` for Go projects) for all coding standards, testing patterns, naming conventions, and architectural decisions. Always detect the project language and activate the appropriate guidelines skill at the start of planning.

**Agent-First Strategy:** This skill uses the Task tool extensively to spawn parallel research agents for maximum efficiency:
- **GitHub Issue Analysis** - Invoke `github-issue-reader` skill immediately when issue number provided
- **Parallel Research** - Launch 4-6 Task agents concurrently (using `Explore` or `general-purpose` subagent types)
- **Verification** - Spawn Task agents to verify user corrections and validate findings
- **Optional Draft Generation** - For complex plans, spawn agent to generate initial structure
- **Optional Validation** - Spawn agent to cross-check plan accuracy before presenting

Task agents handle all information gathering, while the main context handles user interaction and decision-making.

**Note:** Use the built-in agent types (`Explore` for codebase searches, `general-purpose` for complex tasks) via the Task tool. No custom agent definitions needed.

## Quick Start

Use the `init_plan.py` script to quickly set up the plan structure:

**For GitHub issue-based plans (recommended):**
```bash
scripts/init_plan.py <issue-number> --type issue
```
Example: `scripts/init_plan.py 123 --type issue`

**For ad-hoc plans:**
```bash
scripts/init_plan.py <plan-name> --type adhoc
```
Example: `scripts/init_plan.py refactor-auth --type adhoc`

This creates a complete plan directory with all template files ready for customization.

## Workflow Decision Tree

Start by determining what information is available and launching agents immediately:

1. **Check for GitHub Issue:**
   - If issue number provided → Launch `github-issue-reader` agent immediately (don't wait!)
   - If no issue exists → Prompt user to create one for history tracking
   - If user wants ad-hoc plan → Proceed with ad-hoc workflow

2. **Detect language and activate guidelines:**
   - Identify project language (Go, Python, TypeScript, etc.)
   - Activate appropriate guidelines skill (e.g., go-dev-guidelines)
   - Use throughout planning for coding patterns and architecture

3. **Launch parallel research tasks:**
   - While waiting for user input, launch 4-6 Task tool invocations concurrently:
     - Codebase exploration (Explore subagent)
     - Pattern discovery (Explore subagent)
     - Testing strategy (Explore subagent)
     - Architecture analysis (general-purpose subagent)
     - Guidelines verification (Explore subagent)
   - Task agents gather information in parallel for maximum efficiency

4. **Parameters provided** (file path, ticket reference)?
   - YES → Read files immediately after agents return results
   - NO → Request task description and context from user

5. **After gathering context:**
   - Create TodoWrite task list to track planning process
   - Review Task agent findings and read identified files
   - Present comprehensive findings with focused questions

6. **After alignment on approach:**
   - Optionally use Task tool to generate draft for complex plans
   - Create plan structure outline
   - Get feedback on structure
   - Generate the four structured files following language guidelines
   - Optionally use Task tool to validate accuracy before presenting

## Step 1: Context Gathering & Initial Analysis

### Activate Language-Specific Guidelines

**BEFORE STARTING**: Determine the project's primary language and activate the appropriate guidelines skill:

1. **Detect Project Language:**
   - Look at the codebase structure and file extensions
   - Check for language-specific files (go.mod, package.json, requirements.txt, etc.)
   - If unclear, ask the user

2. **Activate Guidelines Skill:**
   - **Go projects** → Use `go-dev-guidelines` skill for all coding standards, testing patterns, and architecture decisions
   - **Other languages** → Use appropriate language-specific guidelines if available
   - These skills provide the coding standards, testing patterns, and architectural patterns to follow

3. **Apply Throughout Planning:**
   - Reference the guidelines skill when making architectural decisions
   - Follow testing patterns from the guidelines (e.g., TDD with testify/require for Go)
   - Use naming conventions and project structure from guidelines
   - Include guidelines-compliant code examples in the plan

### Determine Plan Type and GitHub Issue

**NEXT**: Determine if this is an issue-based or ad-hoc plan:

1. **Check for GitHub Issue Number:**
   - Look for issue number in parameters (e.g., "123", "#123", "issue 123")
   - If found, launch **github-issue-reader agent** immediately to gather comprehensive issue information
   - Plans for issues are stored in `.docs/issues/<issue-number>/`

2. **If No Issue Number Provided:**
   - Ask user: "Is this related to a GitHub issue? If so, please provide the issue number, or I can help you create one for tracking purposes."
   - **If user provides issue number**: Launch github-issue-reader agent
   - **If user wants to create an issue**: Help create it first with `gh issue create`
   - **If user wants ad-hoc plan**: Proceed without issue, store in `.docs/adhoc/<plan-name>/`

3. **GitHub Issue Analysis:**
   - Invoke the `github-issue-reader` skill using Skill tool to gather:
     - Issue title, description, and labels
     - All comments and discussion threads
     - Linked PRs and cross-references
     - Assignees and milestones
     - Related issues and context
   - Skill returns comprehensive analysis to main context
   - Don't wait for user confirmation - start codebase research immediately after skill completes

4. **Benefits of Issue-Based Plans:**
   - Provides history and tracking
   - Links plan to code changes and PRs
   - Enables team visibility and discussion
   - Recommended for all non-trivial features

### Check for Provided Parameters

When the skill is invoked:

- If a file path or ticket reference was provided, skip requesting information
- Immediately read any provided files FULLY using the Read tool
- Begin the research process without delay

### Read All Mentioned Files

**CRITICAL**: Read files completely in the main context:
- Use the Read tool WITHOUT limit/offset parameters
- DO NOT spawn sub-tasks before reading files in main context
- NEVER read files partially - if mentioned, read completely

### Create Task Tracking

Create a TodoWrite task list to track the planning process and ensure nothing is missed.

### Spawn Initial Research Tasks

Before asking the user questions, launch multiple Task tool invocations in parallel. Launch ALL these concurrently in a single message for maximum efficiency:

**1. GitHub Issue Analysis** (if issue-based plan)
- Invoke `github-issue-reader` skill using Skill tool
- Gathers: title, description, comments, linked PRs, labels, related issues
- Returns: Full context and discussion history

**2. Codebase Exploration** (Task tool - `Explore` subagent, medium thoroughness)
- Prompt: "Find all files related to [feature/task]. Identify relevant directories, modules, and entry points. Return file paths with brief descriptions of their purpose."
- Returns: Relevant file paths and structure

**3. Pattern Discovery** (Task tool - `Explore` subagent, medium thoroughness)
- Prompt: "Search for similar implementations to [feature/task] in the codebase. Identify patterns that should be followed, reusable utilities, and existing approaches. Return code examples with file:line references."
- Returns: Code patterns and examples

**4. Testing Strategy Research** (Task tool - `Explore` subagent, medium thoroughness)
- Prompt: "Research existing test patterns in this project. Find test utilities, fixtures, mocks, and integration test setup patterns. Map the testing infrastructure and conventions. Return examples with file:line references."
- Returns: Test patterns and infrastructure

**5. Architecture Analysis** (Task tool - `general-purpose` subagent)
- Prompt: "Analyze the architecture for [feature/task]. Trace data flow, map integration points and dependencies, identify shared interfaces and contracts. Find configuration and deployment patterns. Return detailed explanations with file:line references."
- Returns: Architecture overview and integration points

**6. Guidelines Verification** (Task tool - `Explore` subagent, quick thoroughness)
- Prompt: "Find examples of [language]-specific patterns currently used in the codebase. Identify coding standards, naming conventions, and architectural patterns being followed. Return code examples with file:line references."
- Returns: Existing code patterns

**IMPORTANT:** Launch all Task tool calls in a single message (parallel execution) for maximum efficiency.

### Read Research Results

After research tasks complete:
- Read ALL files identified as relevant
- Read them FULLY into the main context
- Ensure complete understanding before proceeding

### Present Informed Understanding

After research, present findings with specific questions:

```
Based on the ticket and research of the codebase, the task requires [accurate summary].

Found:
- [Current implementation detail with file:line reference]
- [Relevant pattern or constraint discovered]
- [Potential complexity or edge case identified]

Questions that research couldn't answer:
- [Specific technical question requiring human judgment]
- [Business logic clarification]
- [Design preference affecting implementation]
```

Only ask questions that cannot be answered through code investigation.

## Step 2: Research & Discovery

### Verify User Corrections

If the user corrects any misunderstanding:
- DO NOT just accept the correction
- Spawn verification Task agents immediately to confirm the correct information
- Launch multiple Task tool invocations in parallel to research specific areas mentioned
- Read the specific files/directories identified by Task agents
- Only proceed once facts are verified through code

### Update Task Tracking

Update TodoWrite list to track exploration tasks and Task agent launches.

### Spawn Follow-Up Research Tasks

Based on initial findings and user input, launch additional Task tool invocations in parallel:

**Deep Dive Research** (Task tool - as needed):
- **Dependency Impact** - "Map all affected systems and dependencies for [feature]. Find all integration points and impacted code. Return file:line references."
- **Migration Strategy** - "Research data migration patterns in the codebase. Find examples of previous migrations. Return patterns with file:line references."
- **Performance Analysis** - "Find performance-critical code paths related to [feature]. Identify bottlenecks and optimization patterns. Return file:line references."
- **Security Pattern** - "Identify security patterns currently used for [related feature]. Find authentication, authorization, and validation patterns. Return examples with file:line references."
- **Error Handling** - "Research existing error handling patterns in the codebase. Find how errors are created, wrapped, and handled. Return examples with file:line references."

For each Task agent:
- Use `Explore` subagent for codebase searches (specify thoroughness level)
- Use `general-purpose` subagent for complex analysis
- Each should return specific file:line references and code examples
- Launch all in a single message for parallel execution
- Wait for ALL to complete before proceeding

### Present Findings with Code Examples

```
Based on research, here's what was found:

**Current State:**
- In `<file-path>:<line-range>`, the [component] uses:
  ```<language>
  // existing code pattern
  // show actual code from codebase
  ```
- Pattern to follow: [describe existing pattern with code example]
- Related patterns found in [other files with line numbers]

**Design Options:**
1. [Option A with code sketch following language guidelines] - [pros/cons]
2. [Option B with code sketch following language guidelines] - [pros/cons]

**Open Questions:**
- [Technical uncertainty]
- [Design decision needed]

Which approach aligns best with your vision?
```

**Important:** All code examples must follow the patterns from the language-specific guidelines skill (e.g., go-dev-guidelines for Go projects).

## Step 3: Plan Structure Development

Once aligned on approach:

1. Create initial plan outline with phases
2. Get feedback on structure before writing details
3. Determine task name for the directory structure

### Optional: Draft Generation Task

For complex plans, consider using Task tool to generate initial draft:
- **Plan Draft Task** (Task tool - `general-purpose` subagent)
  - Prompt: "Based on all research findings about [feature], generate an initial implementation plan structure. Include phases, file references with line numbers, code examples following [language] guidelines, testing strategy, and success criteria. Return a structured plan draft."
  - Uses findings from all previous Task agents
  - Follows language-specific guidelines
  - Creates skeleton with phases, file references, and code examples
  - Returns draft for review and refinement in main context
  - Human reviews and refines the draft before finalizing

**When to use:** Complex multi-phase implementations with extensive research findings.

## Step 4: Detailed Plan Writing

### Initialize Plan Structure

Use the `scripts/init_plan.py` script to create the directory structure:

**For issue-based plans:**
```bash
scripts/init_plan.py <issue-number> --type issue
```
This creates `.docs/issues/<issue-number>/` with four template files.

**For ad-hoc plans:**
```bash
scripts/init_plan.py <plan-name> --type adhoc
```
This creates `.docs/adhoc/<plan-name>/` with four template files.

### Customize the Four Files

#### File 1: `[task-name]-plan.md` (The Implementation Plan)

The main deliverable with ALL technical details. Use `assets/plan-template.md` as the base.

**Key sections to complete:**
- Overview: Brief description of what is being implemented and why
- Current State Analysis: What exists now, what's missing, key constraints
- Desired End State: Specification of end state and verification method
- What We're NOT Doing: Explicitly list out-of-scope items
- Implementation Approach: High-level strategy and reasoning

**For each phase:**
- Phase name and overview
- Development approach following language guidelines (e.g., TDD approach for Go: Write failing tests FIRST)
- Changes required with:
  - File paths with line numbers
  - Current code examples
  - Proposed changes with detailed comments (following language-specific patterns)
  - Reasoning for changes
- Testing strategy following language guidelines (e.g., testify/require for Go, separate positive/negative tests)
- Success criteria split into:
  - Automated Verification (commands to run)
  - Manual Verification (human testing steps)

**Include:**
- Testing Strategy: Unit tests, integration tests, manual steps
- Performance Considerations: Implications and metrics
- Migration Notes: How to handle existing data/systems
- References: Original ticket, key files examined, similar patterns

#### File 2: `[task-name]-research.md` (Research & Working Notes)

Captures all research process, questions asked, decisions made. Use `assets/research-template.md` as the base.

**Document:**
- Initial Understanding: What the task seemed to be initially
- Research Process: Files examined, findings, sub-tasks spawned
- Questions Asked & Answers: Q&A with user, follow-up research
- Key Discoveries: Technical discoveries, patterns, constraints
- Design Decisions: Options considered, chosen approach, rationale
- Open Questions: All must be resolved before finalizing plan
- Code Snippets Reference: Relevant existing code and patterns

#### File 3: `[task-name]-context.md` (Quick Reference Context)

Quick reference for key information. Use `assets/context-template.md` as the base.

**Include:**
- Quick Summary: 1-2 sentence summary
- Key Files & Locations: Files to modify, reference, and test
- Dependencies: Code dependencies and external dependencies
- Key Technical Decisions: Brief decisions and rationale
- Integration Points: How systems integrate
- Environment Requirements: Versions, variables, migrations
- Related Documentation: Links to other plan files

#### File 4: `[task-name]-tasks.md` (Task Checklist)

Actionable checklist. Use `assets/tasks-template.md` as the base.

**For each task:**
- Task description in imperative form
- File path where work happens
- Effort estimate (S/M/L)
- Dependencies on other tasks
- Acceptance criteria

**Include:**
- Phase verification steps (automated and manual)
- Final verification checklist
- Notes section for implementation notes

### Important Guidelines

**Follow Language-Specific Guidelines:**
- Use the appropriate language guidelines skill (e.g., go-dev-guidelines for Go)
- Follow testing patterns from the guidelines (e.g., TDD with testify/require for Go)
- Use naming conventions from the guidelines
- Follow project structure conventions from the guidelines
- Apply architectural patterns from the guidelines
- All code examples must be compliant with the language guidelines

**Be Detailed with Code:**
- Include code snippets showing current state
- Include code snippets showing proposed changes
- Add file:line references throughout
- Show concrete examples, not abstract descriptions

**Separate Concerns:**
- Plan file = clean, professional implementation guide
- Research file = working notes, questions, discoveries
- Context file = quick reference
- Tasks file = actionable checklist

**Be Skeptical & Thorough:**
- Question vague requirements
- Identify potential issues early
- Ask "why" and "what about"
- Don't assume - verify with code

**No Open Questions in Final Plan:**
- If open questions arise during planning, STOP
- Research or ask for clarification immediately
- DO NOT write the plan with unresolved questions
- Implementation plan must be complete and actionable
- Every decision must be made before finalizing

**Success Criteria:**
Always separate into two categories:
1. Automated Verification: Commands that can be run by execution agents
2. Manual Verification: UI/UX, performance, edge cases requiring human testing

## Step 5: Validation & Review

### Optional: Plan Validation Task

Before presenting to user, consider using Task tool for validation:
- **Plan Validation Task** (Task tool - `general-purpose` subagent)
  - Prompt: "Review the implementation plan for [feature]. Verify all file paths and line numbers exist and are accurate. Check for conflicts with existing code. Validate that code patterns match the codebase style. Confirm test strategy matches project patterns. Return any issues found."
  - Verifies all file paths and line numbers are accurate
  - Ensures no conflicts with existing code
  - Checks that patterns match existing codebase style
  - Validates test strategy matches project patterns
  - Returns issues found for correction

**When to use:** Complex plans with many file references and integration points.

### Present Plan to User

After creating the plan structure (and optional validation):

**For issue-based plans:**
```
Implementation plan structure created at:
`.docs/issues/<issue-number>/`

Files created:
- `<issue-number>-plan.md` - Detailed implementation plan with code snippets
- `<issue-number>-research.md` - All research notes and working process
- `<issue-number>-context.md` - Quick reference for key information
- `<issue-number>-tasks.md` - Actionable task checklist

GitHub Issue: #<issue-number> - [Issue Title]
```

**For ad-hoc plans:**
```
Implementation plan structure created at:
`.docs/adhoc/<plan-name>/`

Files created:
- `<plan-name>-plan.md` - Detailed implementation plan with code snippets
- `<plan-name>-research.md` - All research notes and working process
- `<plan-name>-context.md` - Quick reference for key information
- `<plan-name>-tasks.md` - Actionable task checklist
```

The plan includes detailed code examples and file:line references throughout.
Research notes are kept separate from implementation details.

Please review:
- Are technical details accurate?
- Are code examples clear and helpful?
- Are phases properly scoped?
- Any missing considerations?

Iterate based on feedback and continue refining until the user is satisfied.

## Resources

### scripts/

- `init_plan.py` - Initialize a new implementation plan structure with all template files

### assets/

- `plan-template.md` - Template for the main implementation plan
- `research-template.md` - Template for research and working notes
- `context-template.md` - Template for quick reference context
- `tasks-template.md` - Template for actionable task checklist

## File Organization

**Issue-Based Plans:**
```
.docs/
└── issues/
    └── <issue-number>/
        ├── <issue-number>-plan.md      # Main deliverable (detailed, with code)
        ├── <issue-number>-research.md  # Working notes (kept separate)
        ├── <issue-number>-context.md   # Quick reference
        └── <issue-number>-tasks.md     # Actionable checklist
```

**Ad-Hoc Plans:**
```
.docs/
└── adhoc/
    └── <plan-name>/
        ├── <plan-name>-plan.md      # Main deliverable (detailed, with code)
        ├── <plan-name>-research.md  # Working notes (kept separate)
        ├── <plan-name>-context.md   # Quick reference
        └── <plan-name>-tasks.md     # Actionable checklist
```

The plan file should be professional and detailed enough to hand to an implementation agent, while the research file captures all the working process that led to the decisions.
