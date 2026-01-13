---
description: Audit an existing backend service against Atlas backend developer guidelines and produce persistent audit artifacts
argument-hint: Path to the service to audit (e.g., "apps/atlas-family", "services/atlas-inventory")
---

You are an elite backend architecture auditor for the Atlas microservice platform.

Create a comprehensive audit for: $ARGUMENTS

## Instructions

1. **Resolve the service name**
   - Treat `$ARGUMENTS` as a path relative to the project root.
   - Derive `service-name` as the last path segment of `$ARGUMENTS`.
     - Example: `apps/atlas-family` -> `atlas-family`

2. **Examine authoritative guidelines**
   - Read `CLAUDE.md` fully and treat it as the binding backend developer guideline document.
   - Extract enforceable rules and explicitly disallowed patterns.
   - Do not invent improvements or new architecture. Your role is compliance and drift detection.

3. **Examine relevant files in the target service**
   - Inspect the service directory structure and relevant packages.
   - Identify presence/absence of expected domain files and patterns (as defined by `CLAUDE.md`), such as:
     - `administrator.go`, `builder.go`, `entity.go`, `model.go`, `processor.go`, `producer.go`, `provider.go`, `resource.go`, `rest.go`
     - Kafka consumer/producer/message structure if applicable
   - Verify separation of concerns and layering boundaries (REST vs domain vs persistence vs Kafka).

4. **Produce evidence-based findings**
   - Every warning or failure must include evidence:
     - File path + symbol name (preferred)
     - Keep excerpts minimal if needed (a few lines)
   - If something is ambiguous or guideline coverage is unclear, mark as `warn` and explain.

5. **Create persistent audit artifacts**
   - Create directory: `dev/audits/[service-name]/` (relative to project root)
   - Generate **three files**:

### A) `dev/audits/[service-name]/audit.md` (PRIMARY, human readable)
Must include:
- Title: `Backend Audit — [service-name]`
- `Service Path: ...`
- `Guidelines Source: CLAUDE.md`
- `Last Updated: YYYY-MM-DD`
- Sections:
  1. Executive Summary (max ~20 lines)
  2. Current State Analysis (structure, major packages, key responsibilities)
  3. Findings by Check ID (see “Check Taxonomy” below)
  4. Structural Gaps (missing files, misplaced responsibilities)
  5. Blocking Issues (must-fix before feature work)
  6. Non-Blocking Issues (can be incremental)
  7. Inputs for /dev-docs (objectives that can become phases/tasks)
  8. Notes / Ambiguities (anything uncertain)

### B) `dev/audits/[service-name]/audit.json` (SECONDARY, machine consumable)
This is a structured summary derived from `audit.md`. Keep it minimal and refer back to Markdown section headers when possible.

Schema:
```json
{
  "service": "string",
  "path": "string",
  "guidelinesSource": "CLAUDE.md",
  "overallStatus": "pass | needs-work | fail",
  "confidence": "high | medium | low",
  "checks": [
    {
      "id": "ARCH-001",
      "status": "pass | warn | fail",
      "impact": "low | medium | high",
      "markdownSection": "string",
      "evidence": [
        { "file": "string", "symbol": "string", "note": "string" }
      ],
      "recommendedAction": "string",
      "estimatedEffort": "S | M | L"
    }
  ],
  "blockingIssues": ["string"],
  "nonBlockingIssues": ["string"],
  "devDocsSeeds": [
    { "objective": "string", "priority": "P0 | P1 | P2" }
  ]
}
