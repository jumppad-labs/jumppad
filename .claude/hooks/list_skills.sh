#!/bin/bash

# Hook script to list available skills for Claude Code
# This script is executed at session start to remind Claude of available skills

# Get the git root directory (if we're in a git repo)
GIT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null)

# Define skill directories to check (project first, then user)
SKILL_DIRS=()
if [ -n "$GIT_ROOT" ] && [ -d "$GIT_ROOT/.claude/skills" ]; then
    SKILL_DIRS+=("$GIT_ROOT/.claude/skills")
fi
if [ -d "$HOME/.claude/skills" ]; then
    SKILL_DIRS+=("$HOME/.claude/skills")
fi

# Exit if no skill directories exist
if [ ${#SKILL_DIRS[@]} -eq 0 ]; then
    exit 0
fi

# Count total skills across all directories
SKILL_COUNT=0
for dir in "${SKILL_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        count=$(find "$dir" -maxdepth 1 -type d ! -path "$dir" 2>/dev/null | wc -l)
        SKILL_COUNT=$((SKILL_COUNT + count))
    fi
done

if [ "$SKILL_COUNT" -eq 0 ]; then
    exit 0
fi

# Output reminder message
echo "Available Skills (${SKILL_COUNT}):"
echo ""

# Iterate through each skill directory in all locations
for SKILLS_DIR in "${SKILL_DIRS[@]}"; do
    for skill_dir in "$SKILLS_DIR"/*; do
    if [ -d "$skill_dir" ]; then
        skill_name=$(basename "$skill_dir")
        skill_md="$skill_dir/SKILL.md"

        if [ -f "$skill_md" ]; then
            # Extract description from YAML frontmatter (handle multi-line descriptions)
            description=$(awk '
                /^---$/ { in_fm = !in_fm; next }
                in_fm && /^description:/ {
                    sub(/^description: */, "")
                    desc = $0
                    while (getline > 0 && !/^[a-z].*:/) {
                        if (/^---$/) break
                        desc = desc " " $0
                    }
                    print desc
                    exit
                }
            ' "$skill_md")

            if [ -n "$description" ]; then
                echo "- ${skill_name}: ${description}"
            else
                echo "- ${skill_name}"
            fi
        else
            echo "- ${skill_name} (no SKILL.md found)"
        fi
    fi
    done
done

echo ""
