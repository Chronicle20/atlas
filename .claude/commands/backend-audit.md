---
description: Adversarially audit a Go service against backend developer guidelines — dispatches the backend-guidelines-reviewer agent
argument-hint: Path to the service to audit (e.g., "services/atlas-account")
---

Dispatch the `backend-guidelines-reviewer` agent against: **$ARGUMENTS**.

Pass the service path so the agent can run the build/test gate, the DOM-* / SUB-* checklists, and (if auth-related) SEC-* checks. The agent writes `audit.md` and `audit.json` under `docs/audits/<service-name>/` (or under the active task folder if invoked from a feature branch with a `plan.md`).

After the agent completes, summarize PASS / NEEDS-WORK / FAIL status and any blocking items.
