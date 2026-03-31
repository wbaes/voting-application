# Agent Behavior Guidelines

This document outlines the expected behavior and operational guidelines for agents working in this project.

## Core Principles

### 1. Skill Utilization

- **Always leverage available skills** when they match the task at hand
- Skills provide specialized capabilities and domain knowledge
- Check available skills before implementing solutions manually
- Prefer skill-based approaches over manual implementation when applicable

### 2. Step-by-Step Planning

- **Break down complex tasks** into clear, manageable steps
- Create structured implementation plans for non-trivial work
- Document the approach before executing
- Use the session workspace (`plan.md`) for planning when needed
- Track progress using the SQL todo system for multi-step tasks

### 3. User Confirmation

- **Request user confirmation** before executing non-trivial tasks
- Non-trivial tasks include:
  - Making significant code changes that affect core functionality
  - Deleting or modifying multiple files
  - Implementing new features or major refactors
  - Changes that could impact production behavior
  - Installing new dependencies or tools
  - Making architectural decisions
- Provide a clear summary of what will be done and ask for approval
- For trivial tasks (typos, minor fixes, single-line changes), proceed without confirmation

### 4. Self-Documentation

- **Expand this AGENTS.md file** when:
  - New patterns or best practices emerge
  - Project-specific conventions are established
  - Common pitfalls or gotchas are discovered
  - New workflows or processes are introduced
- Keep this document up-to-date as the living source of truth for agent behavior
- Document lessons learned and refinements to the development process

## Workflow Best Practices

### Before Starting Work

1. Understand the full scope of the request
2. Check for available skills that could help
3. For non-trivial tasks: create a plan and get user confirmation
4. Identify dependencies and potential impacts

### During Execution

1. Follow the planned steps systematically
2. Update todo status as work progresses
3. Validate changes incrementally
4. Report progress clearly and concisely

### After Completion

1. Verify the expected outcome
2. Run relevant tests or validations
3. Clean up temporary files or artifacts
4. Summarize what was done

## Project-Specific Guidelines

### Tech Stack

Expand this once the stack has been decided
### Project Structure

Expand this once the structure becomes clear

### Development

Expand this once the stack has been decided

---

_This document is maintained by agents and should be updated whenever new behavioral patterns or guidelines are established._
