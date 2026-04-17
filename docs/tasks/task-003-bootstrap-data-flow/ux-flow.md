---
name: Bootstrap Data Flow — UX Flow
description: User-visible behavior of the /setup page changes — state machine, gating matrix, toasts, and accessibility notes.
type: ux-flow
task: task-003-bootstrap-data-flow
---

# Bootstrap Data Flow — UX Flow

Documents the user-visible behavior of the `/setup` page changes. Intended audience: the implementer and the reviewer.

## Layout

The existing "Bootstrap" section gains a "Game Data" card above the current seed-action grid. The card has three rows, each with a label, status badge, and action button:

| Row | Label | Badge | Button |
|---|---|---|---|
| 1 | Upload WZ | `N .wz files, N MB` | "Upload" (opens file picker) |
| 2 | Extract | `N XMLs extracted` | "Run Extraction" |
| 3 | Ingest | `N documents loaded` | "Process Data" |

The seven existing seed buttons are unchanged.

## State machine

Each row's button has three possible states:

- **Disabled — prerequisite not met.** Tooltip: "Upload WZ files first" / "Run extraction first".
- **Disabled — busy.** Spinner in the button; tooltip: "Uploading…" / "Extracting…" / "Ingesting…".
- **Enabled.** Click fires the mutation.

Gating matrix:

| Button | Disabled when |
|---|---|
| Upload | upload mutation pending |
| Run Extraction | `wzInputStatus.fileCount == 0` OR any of {upload, extract, ingest} mutation pending |
| Process Data | `extractionStatus.fileCount == 0` OR extract/ingest mutation pending |

Upload does not gate on extraction state — a fresh upload is always allowed (subject to the 409 on concurrent tenant busy, which is surfaced as a toast rather than gating the button client-side, since the client can't distinguish "another tab is uploading" without a status read).

## Status polling

Three react-query hooks, `staleTime: 0`, `refetchInterval: 5000`, enabled only while the page is mounted:

- `useWzInputStatus` — `GET /api/wz/input`
- `useExtractionStatus` — `GET /api/wz/extractions`
- `useDataStatus` — `GET /api/data/status`

After a successful mutation, `queryClient.invalidateQueries` on the relevant key so the downstream button re-gates within one paint rather than waiting for the next 5 s tick. Invalidation map:

| Successful mutation | Invalidates |
|---|---|
| `useUploadWzFiles` | `wzInputStatus`, `extractionStatus` (for the stale-warning comparison) |
| `useRunWzExtraction` | `extractionStatus`, `dataStatus` |
| `useRunDataProcessing` | `dataStatus` |

## Stale-extraction warning

When `wzInputStatus.updatedAt` is non-null AND `extractionStatus.updatedAt` is non-null AND `wzInputStatus.updatedAt > extractionStatus.updatedAt`, render a yellow inline warning under the Ingest row:

> ⚠️ Uploaded WZ files are newer than the last extraction. Re-run extraction before ingest to avoid stale data.

The warning does NOT disable the Ingest button — operators may still choose to ingest older XMLs (e.g., if they just re-uploaded to stage a future extraction). It's informational.

## Toasts

- Upload start: none (the file picker itself is the acknowledgement).
- Upload success: `"WZ files uploaded (<N> files, <size>)"`.
- Upload 400: `"Upload rejected: <reason>"` (reason from response body).
- Upload 409: `"Another upload or extraction is in progress for this tenant. Try again in a moment."`.
- Extract success: `"Extraction complete"` — file count comes from the next status poll, not the trigger response.
- Extract failure: `"Extraction failed: <reason>"`.
- Ingest success: `"Data processing started"` — ingest is async; the `documentCount` badge updates as workers complete.
- Ingest failure: `"Data processing failed: <reason>"`.

## Badge formatting

- "12 .wz files" / "1 .wz file" (pluralization aware).
- Byte totals use `Intl.NumberFormat` with B / KB / MB / GB boundaries: `1.2 GB`.
- Document counts use thousands separators: `18,204 documents`.
- While the status query is pending (no data yet), show a neutral em-dash: "—".

## Accessibility

- File input uses native `<input type="file" accept=".zip">` with visible label.
- Buttons are keyboard-navigable in the order Upload → Extract → Ingest.
- Badges use `aria-live="polite"` so a screen reader announces count changes after a mutation.
- The stale-extraction warning uses `role="status"` and is positioned before the Ingest button so screen readers encounter it in reading order.
