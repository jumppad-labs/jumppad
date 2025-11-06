#!/usr/bin/env python3
"""
Initialize a new implementation plan structure.

This script creates the directory structure and initial files for a new
implementation plan, using templates from the assets directory.

Plans can be either issue-based (linked to GitHub issues) or ad-hoc.
"""

import os
import sys
import argparse
from datetime import datetime
from pathlib import Path


def get_template_path(template_name):
    """Get the path to a template file."""
    script_dir = Path(__file__).parent
    skill_dir = script_dir.parent
    return skill_dir / "assets" / template_name


def replace_placeholders(content, replacements):
    """Replace placeholders in content with actual values."""
    for placeholder, value in replacements.items():
        content = content.replace(f"{{{{{placeholder}}}}}", value)
    return content


def init_plan(plan_id, plan_type="issue"):
    """
    Initialize a new implementation plan structure.

    Args:
        plan_id: The issue number (for issue plans) or plan name (for adhoc plans)
        plan_type: Either "issue" or "adhoc" (default: issue)
    """
    # Determine base directory based on plan type
    if plan_type == "issue":
        base_dir = Path(".docs") / "issues"
    elif plan_type == "adhoc":
        base_dir = Path(".docs") / "adhoc"
    else:
        raise ValueError(f"Invalid plan type: {plan_type}. Must be 'issue' or 'adhoc'")

    # Create the plan directory
    plan_dir = base_dir / str(plan_id)
    plan_dir.mkdir(parents=True, exist_ok=True)

    # Prepare replacements for templates
    now = datetime.now()

    # Determine display name based on plan type
    if plan_type == "issue":
        display_name = f"Issue #{plan_id}"
    else:
        display_name = str(plan_id).replace("-", " ").title()

    replacements = {
        "TASK_NAME": display_name,
        "CREATED_DATE": now.strftime("%Y-%m-%d %H:%M"),
        "UPDATED_DATE": now.strftime("%Y-%m-%d %H:%M"),
        "RESEARCH_DATE": now.strftime("%Y-%m-%d"),
        "USER_NAME": os.environ.get("USER", "User"),
        "OVERVIEW": "[Brief description of what we're implementing and why]",
        "CURRENT_STATE": "[What exists now, what's missing, key constraints discovered]",
        "DESIRED_STATE": "[Specification of the desired end state after this plan is complete, and how to verify it]",
        "OUT_OF_SCOPE": "[Explicitly list out-of-scope items to prevent scope creep]",
        "APPROACH": "[High-level strategy and reasoning]",
        "PHASE_1_NAME": "Foundation",
        "PHASE_1_OVERVIEW": "[What this phase accomplishes]",
        "COMPONENT_1": "Component/File Group",
        "CHANGES_DESCRIPTION": "[Detailed description of changes]",
        "REASONING": "[Why this change is needed]",
        "PERFORMANCE_NOTES": "[Any performance implications with specific metrics or code patterns]",
        "MIGRATION_NOTES": "[If applicable, how to handle existing data/systems with examples]",
        "INITIAL_UNDERSTANDING": "[What we thought the task was about initially]",
        "DECISION_1_TOPIC": "Decision Topic",
        "DECISION_2_TOPIC": "Another Decision",
        "QUICK_SUMMARY": "[1-2 sentence summary of what this task does]",
        "DECISION_1": "Decision Topic",
        "DECISION_2": "Another Decision Topic",
        "TICKET_PATH": f"GitHub Issue #{plan_id}" if plan_type == "issue" else "N/A",
        "PHASE_2_NAME": "Next Phase",
    }

    # Define the files to create
    files = {
        f"{plan_id}-plan.md": "plan-template.md",
        f"{plan_id}-research.md": "research-template.md",
        f"{plan_id}-context.md": "context-template.md",
        f"{plan_id}-tasks.md": "tasks-template.md",
    }

    # Create each file from its template
    created_files = []
    for filename, template_name in files.items():
        template_path = get_template_path(template_name)
        output_path = plan_dir / filename

        # Read template
        with open(template_path, "r") as f:
            content = f.read()

        # Replace placeholders
        content = replace_placeholders(content, replacements)

        # Write output file
        with open(output_path, "w") as f:
            f.write(content)

        created_files.append(str(output_path))

    return plan_dir, created_files


def main():
    parser = argparse.ArgumentParser(
        description="Initialize a new implementation plan structure"
    )
    parser.add_argument(
        "plan_id",
        help="Issue number (for issue plans) or plan name (for adhoc plans)"
    )
    parser.add_argument(
        "--type",
        choices=["issue", "adhoc"],
        default="issue",
        help="Type of plan: 'issue' for GitHub issue-based plans, 'adhoc' for standalone plans (default: issue)"
    )

    args = parser.parse_args()

    try:
        plan_dir, created_files = init_plan(args.plan_id, args.type)

        print(f"✅ Implementation plan initialized successfully!")
        print(f"\nPlan type: {args.type}")
        print(f"Plan directory: {plan_dir}")
        print(f"\nFiles created:")
        for file_path in created_files:
            print(f"  - {file_path}")
        print(f"\nNext steps:")
        print(f"  1. Review and customize the generated files")
        print(f"  2. Begin research and update {args.plan_id}-research.md")
        print(f"  3. Fill in the implementation details in {args.plan_id}-plan.md")

        if args.type == "issue":
            print(f"\nLinked to GitHub Issue #{args.plan_id}")

    except Exception as e:
        print(f"❌ Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
