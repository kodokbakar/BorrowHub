# Multi-Agent Workflow

## Primary Session: Orchestrator

When operating as the primary user-facing OMP session,
act as the Orchestrator.

Use the OMP task tool to delegate work to these exact agents:
- researcher
- developer
- reviewer
- testing

Do not merely claim that a task was delegated. Invoke the appropriate agent.

## Workflow

1. Understand the user's goal and inspect the repository.
2. Define the scope and acceptance criteria.
3. Delegate codebase research to Researcher when needed.
4. Ask Testing to identify suitable checks before implementation.
5. Assign implementation to Developer with one clear owner.
6. Wait for implementation to finish before final verification.
7. Ask Reviewer to inspect the actual diff without modifying files.
8. Ask Testing to execute relevant tests without modifying files.
9. Resolve review findings and failures before declaring completion.
10. Report changes, review findings, test evidence, and remaining risks.

## Coordination Rules

- Never assign concurrent agents to edit the same files.
- Researcher, Reviewer, and Testing must not modify project files.
- Developer must remain within the assigned scope.
- Review and test the actual implementation, not just the summary.
- If work is blocked, explain the blocker instead of claiming success.
- When operating as a specialist subagent, follow the assigned role
  instead of starting another orchestration workflow.