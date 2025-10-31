#!/usr/bin/env python3
"""
Fetch comprehensive GitHub issue information using the gh CLI.

This script retrieves:
- Basic issue details (title, body, state, labels, assignees)
- All comments with full threading
- Metadata (labels, milestones, assignees)
- Related items (linked PRs, cross-references)

Requires:
- gh CLI installed and authenticated
- Internet connection
"""

import sys
import json
import subprocess
import re
from typing import Optional, Dict, List, Any


def run_gh_command(cmd: List[str]) -> tuple[bool, str, str]:
    """
    Run a gh CLI command and return success status, stdout, and stderr.

    Args:
        cmd: Command and arguments as a list

    Returns:
        Tuple of (success, stdout, stderr)
    """
    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            check=False
        )
        return result.returncode == 0, result.stdout, result.stderr
    except FileNotFoundError:
        return False, "", "gh CLI not found. Please install GitHub CLI."
    except Exception as e:
        return False, "", str(e)


def parse_issue_reference(ref: str) -> tuple[Optional[str], Optional[str]]:
    """
    Parse an issue reference into repo and issue number.

    Accepts:
    - Issue number: "123"
    - URL: "https://github.com/owner/repo/issues/123"
    - Owner/repo#number: "owner/repo#123"

    Returns:
        Tuple of (repo, issue_number) or (None, issue_number) for current repo
    """
    # Try URL format
    url_match = re.match(r'https?://github\.com/([^/]+/[^/]+)/issues/(\d+)', ref)
    if url_match:
        return url_match.group(1), url_match.group(2)

    # Try owner/repo#number format
    repo_match = re.match(r'([^/]+/[^#]+)#(\d+)', ref)
    if repo_match:
        return repo_match.group(1), repo_match.group(2)

    # Try plain number
    if ref.isdigit():
        return None, ref

    return None, None


def fetch_issue_details(repo: Optional[str], issue_num: str) -> tuple[bool, Dict[str, Any]]:
    """
    Fetch basic issue details using gh issue view.

    Args:
        repo: Repository in owner/repo format, or None for current repo
        issue_num: Issue number

    Returns:
        Tuple of (success, issue_data)
    """
    cmd = ["gh", "issue", "view", issue_num, "--json",
           "number,title,body,state,author,createdAt,updatedAt,closedAt,"
           "labels,assignees,milestone,url,comments"]

    if repo:
        cmd.extend(["-R", repo])

    success, stdout, stderr = run_gh_command(cmd)

    if not success:
        return False, {"error": stderr}

    try:
        return True, json.loads(stdout)
    except json.JSONDecodeError:
        return False, {"error": "Failed to parse issue data"}


def fetch_timeline(repo: Optional[str], issue_num: str) -> List[Dict[str, Any]]:
    """
    Fetch issue timeline events (cross-references, linked PRs, etc.).

    Args:
        repo: Repository in owner/repo format, or None for current repo
        issue_num: Issue number

    Returns:
        List of timeline events
    """
    # Construct the API endpoint
    if repo:
        api_path = f"repos/{repo}/issues/{issue_num}/timeline"
    else:
        # Get current repo
        success, stdout, _ = run_gh_command(["gh", "repo", "view", "--json", "nameWithOwner"])
        if not success:
            return []
        try:
            repo_data = json.loads(stdout)
            repo = repo_data.get("nameWithOwner", "")
            if not repo:
                return []
            api_path = f"repos/{repo}/issues/{issue_num}/timeline"
        except (json.JSONDecodeError, KeyError):
            return []

    cmd = ["gh", "api", api_path, "--paginate"]
    success, stdout, _ = run_gh_command(cmd)

    if not success:
        return []

    try:
        return json.loads(stdout)
    except json.JSONDecodeError:
        return []


def format_issue_markdown(issue_data: Dict[str, Any], timeline: List[Dict[str, Any]]) -> str:
    """
    Format issue data as markdown.

    Args:
        issue_data: Issue data from gh issue view
        timeline: Timeline events from API

    Returns:
        Formatted markdown string
    """
    lines = []

    # Header
    lines.append(f"# Issue #{issue_data['number']}: {issue_data['title']}")
    lines.append("")
    lines.append(f"**URL**: {issue_data['url']}")
    lines.append(f"**State**: {issue_data['state']}")
    lines.append(f"**Author**: {issue_data['author']['login']}")
    lines.append(f"**Created**: {issue_data['createdAt']}")
    lines.append(f"**Updated**: {issue_data['updatedAt']}")

    if issue_data.get('closedAt'):
        lines.append(f"**Closed**: {issue_data['closedAt']}")

    lines.append("")

    # Labels
    if issue_data.get('labels'):
        labels = [label['name'] for label in issue_data['labels']]
        lines.append(f"**Labels**: {', '.join(labels)}")
        lines.append("")

    # Assignees
    if issue_data.get('assignees'):
        assignees = [a['login'] for a in issue_data['assignees']]
        lines.append(f"**Assignees**: {', '.join(assignees)}")
        lines.append("")

    # Milestone
    if issue_data.get('milestone'):
        lines.append(f"**Milestone**: {issue_data['milestone']['title']}")
        lines.append("")

    # Body
    lines.append("## Description")
    lines.append("")
    body = issue_data.get('body', '').strip()
    if body:
        lines.append(body)
    else:
        lines.append("*No description provided*")
    lines.append("")

    # Comments
    if issue_data.get('comments'):
        lines.append(f"## Comments ({len(issue_data['comments'])})")
        lines.append("")

        for i, comment in enumerate(issue_data['comments'], 1):
            lines.append(f"### Comment {i} by {comment['author']['login']} ({comment['createdAt']})")
            lines.append("")
            lines.append(comment['body'])
            lines.append("")

    # Related items from timeline
    cross_refs = []
    linked_prs = []

    for event in timeline:
        event_type = event.get('event', '')

        if event_type == 'cross-referenced':
            source = event.get('source', {})
            if source.get('issue'):
                ref_issue = source['issue']
                cross_refs.append(f"- #{ref_issue.get('number')} - {ref_issue.get('title', 'Unknown')}")

        elif event_type == 'connected':
            subject = event.get('subject', {})
            if subject.get('__typename') == 'PullRequest':
                linked_prs.append(f"- PR #{subject.get('number')} - {subject.get('title', 'Unknown')}")

    if cross_refs:
        lines.append("## Cross-Referenced Issues")
        lines.append("")
        lines.extend(cross_refs)
        lines.append("")

    if linked_prs:
        lines.append("## Linked Pull Requests")
        lines.append("")
        lines.extend(linked_prs)
        lines.append("")

    return "\n".join(lines)


def main():
    """Main entry point."""
    if len(sys.argv) < 2:
        print("Usage: fetch_issue.py <issue-reference>", file=sys.stderr)
        print("", file=sys.stderr)
        print("Examples:", file=sys.stderr)
        print("  fetch_issue.py 123", file=sys.stderr)
        print("  fetch_issue.py owner/repo#123", file=sys.stderr)
        print("  fetch_issue.py https://github.com/owner/repo/issues/123", file=sys.stderr)
        sys.exit(1)

    issue_ref = sys.argv[1]

    # Parse the issue reference
    repo, issue_num = parse_issue_reference(issue_ref)

    if issue_num is None:
        print(f"Error: Invalid issue reference: {issue_ref}", file=sys.stderr)
        sys.exit(1)

    # Fetch issue details
    success, issue_data = fetch_issue_details(repo, issue_num)

    if not success:
        print(f"Error fetching issue: {issue_data.get('error', 'Unknown error')}", file=sys.stderr)
        sys.exit(1)

    # Fetch timeline for related items
    timeline = fetch_timeline(repo, issue_num)

    # Format and output
    markdown = format_issue_markdown(issue_data, timeline)
    print(markdown)


if __name__ == "__main__":
    main()
