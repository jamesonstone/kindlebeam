# AGENTS.md

## Spec Kit is the source of truth

- Constitution: `.specify/memory/constitution.md`
- Feature specs live under: `specs/<feature-branch>/`
  - `spec.md` (requirements)
  - `plan.md` (implementation plan)
  - `tasks.md` (executable task list)
  - optional supporting docs: `research.md`, `data-model.md`, `contracts/`, `quickstart.md`, `implementation-details/`

## Workflow contract (SDD)

- Specs drive code. Code serves specs. [oai_citation:2‡GitHub](https://raw.githubusercontent.com/github/spec-kit/main/spec-driven.md)
- For any change:
  1. locate the relevant feature directory in `specs/<feature-branch>/`
  2. read `spec.md` → `plan.md` → `tasks.md`
  3. implement tasks in order
  4. verify (tests / validation steps from plan/quickstart)
  5. if reality diverges, update `spec.md` / `plan.md` / `tasks.md` first, then code

## Multi-feature rule

- Never mix features in one `specs/<feature-branch>/` directory.
- If work spans features, update each feature’s docs separately.
