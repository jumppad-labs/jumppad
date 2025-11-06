---
name: github-issue-reader
description: Load comprehensive GitHub issue information including title, description, comments, labels, assignees, milestones, and related items (linked PRs and cross-references). This skill should be used when planning to fix an issue, when detailed issue context is needed for implementation work, or when a plan command needs to understand the full scope of an issue.
---

# GitHub Issue Reader

## Overview

Fetch complete GitHub issue information to provide context for planning and implementation. This skill retrieves all relevant data from a GitHub issue including the description, all comments and discussion, metadata (labels, assignees, milestones), and related items (linked pull requests and cross-referenced issues).

## When to Use This Skill

Use this skill when:
- Creating a plan to fix or implement an issue
- Understanding the full context and discussion around an issue
- Gathering requirements from an issue before starting work
- Reviewing an issue's history and related work
- Analyzing cross-references and linked pull requests

## Quick Start

Fetch issue information using the `fetch_issue.py` script with any of these formats:

```bash
# Current repository issue by number
scripts/fetch_issue.py 123

# Specific repository issue
scripts/fetch_issue.py owner/repo#456

# Issue by full URL
scripts/fetch_issue.py https://github.com/owner/repo/issues/789
```

The script outputs formatted markdown containing all issue information.

## Usage Instructions

### Prerequisites

Ensure the GitHub CLI (`gh`) is installed and authenticated:
```bash
gh auth status
```

If not authenticated, run:
```bash
gh auth login
```

### Fetching Issue Information

Execute the `fetch_issue.py` script with an issue reference. The script accepts three formats:

1. **Issue number** (for current repository): `123`
2. **Owner/repo format**: `owner/repo#123`
3. **Full URL**: `https://github.com/owner/repo/issues/123`

The script will fetch and output:
- Issue number, title, and URL
- Current state (open/closed)
- Author and timestamps (created, updated, closed)
- Labels, assignees, and milestone
- Full issue description
- All comments with authors and timestamps
- Cross-referenced issues
- Linked pull requests

### Example Output Structure

```markdown
# Issue #123: Add new feature for X

**URL**: https://github.com/owner/repo/issues/123
**State**: open
**Author**: username
**Created**: 2024-10-15T10:30:00Z
**Labels**: enhancement, priority-high
**Assignees**: developer1, developer2

## Description

[Full issue description]

## Comments (3)

### Comment 1 by user1 (2024-10-16T09:00:00Z)

[Comment body]

## Linked Pull Requests

- PR #124 - Fix for issue #123

## Cross-Referenced Issues

- #120 - Related issue
```

### Integration with Planning Workflows

When using this skill as part of a planning workflow:

1. Invoke the skill with the issue reference
2. Review the complete issue context provided
3. Use the information to create an accurate implementation plan
4. Reference specific comments or requirements from the issue
5. Check linked PRs for existing work or context

### Error Handling

The script handles common errors:
- **gh CLI not found**: Install GitHub CLI from https://cli.github.com
- **Not authenticated**: Run `gh auth login`
- **Issue not found**: Verify the issue number and repository
- **Rate limiting**: Wait and retry, or check GitHub API rate limits

## Resources

### scripts/fetch_issue.py

Python script that uses the GitHub CLI to fetch comprehensive issue data. Can be executed directly without loading into context.

**Usage**: `scripts/fetch_issue.py <issue-reference>`

**Returns**: Formatted markdown with complete issue information

**Requirements**:
- GitHub CLI (`gh`) installed and authenticated
- Python 3.6+
- Internet connection
