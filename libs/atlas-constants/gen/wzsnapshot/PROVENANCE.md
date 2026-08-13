# wzsnapshot provenance (task-187 Task 3)

Drain timestamp: 2026-08-12T18:02Z (kubectl context `bee`, namespace
`atlas-main`, service `atlas-data`, pod `atlas-data-fc5f98fd9-czg6l`, one of
four running replicas -- any replica serves any tenant via header-based
tenant resolution, so the specific pod name is incidental).

## Method

Tenant IDs were re-listed live via `GET /api/tenants` for this re-drain
(task-218 FR-0 Step 1), rather than reused verbatim from task-187 Task 1
(`docs/tasks/task-187-version-aware-id-semantics/audit/README.md`) --
tenant ids can change on reprovision. All 10 ids returned by this listing
were identical to the ones task-187 recorded; the table below is therefore
unchanged from the prior drain.

**Skill list endpoint (`GET /api/data/skills`) is unavailable in this
baseline** -- probed with several parameter shapes
(`?page[size]=5`, `?page[number]=1&page[size]=5`, no query, trailing slash);
every shape returned `HTTP/1.1 400 Bad Request`. Per the task brief's
documented fallback, **all 10 live-tenant snapshots use the jobs-union
method**: drain `GET /api/data/jobs?page[size]=200` for the tenant, take the
job `id` field of every row as the job id-set, and take the union of every
row's `attributes.skills` array as the skill id-set.

## Re-drain 2026-08 (task-218 FR-0)

Re-drained with `tools/wzsnapshot-drain.sh` piped into
`gen/wzsnapshot/cmd/mksnapshot`, which canonicalizes (sort + de-duplicate)
and recomputes the pinned `hash` — so a re-drain is reproducible rather than
a hand edit.

**Method is unchanged** (`GET /api/data/skills` still returns HTTP 400 in
this baseline; the jobs-union fallback is still required). Only the DATA is
fresh. The 2026-07-30 drain predated the Evan job documents being populated
(the `Skill.wz/Dragon/` subdirectory defect), so the union contained zero
`22xxxxxx` skills and every Evan wire id was unbound on every version.
`GET /api/data/jobs/2216/skills` now returns
`[22160000, 22161001, 22161002, 22161003]` on GMS 84/92/95 and JMS 185, and
empty on GMS 48/83 — agreeing exactly with the per-skill availability sweep
in `docs/tasks/task-218-player-cast-mists/design.md` §1.1.

The re-drain necessarily pulls in every other skill the jobs-union now
surfaces that it did not in July; that is inherent to regenerating the
snapshot wholesale. The acceptance gate is therefore that the regenerated
binding diff is ADDITIVE ONLY (no previously-bound wire id changed its
identity), not that the diff is small — task-187's divergence semantics and
the ban list behind `tools/skill-job-id-guard.sh` both depend on those
bindings.

`page[size]=200` returned every job in a single page for all 10 tenants
(`meta.page.last == 1` in every drain -- largest tenant, gms_92/gms_95, has
101 jobs; no tenant required a second page), so no pagination loop was
needed in practice; the brief's `page[size]=...` + `meta.page.last` guidance
was followed but degenerated to one request per tenant.

Command shape (per tenant, run via `kubectl exec` into the atlas-data pod):

```
wget -q -O- \
  --header "TENANT_ID: <tenant-uuid>" \
  --header "REGION: <GMS|JMS>" \
  --header "MAJOR_VERSION: <major>" \
  --header "MINOR_VERSION: 1" \
  "http://localhost:8080/api/data/jobs?page[size]=200"
```

## Per-version tenant IDs and drain method

| file | region | major | minor | tenant id | method |
|---|---|---|---|---|---|
| gms_12_1.json | gms | 12 | 1 | *(none -- see below)* | mirrored from gms_48_1.json |
| gms_48_1.json | gms | 48 | 1 | `e1f06ae2-80c1-47f7-bb6f-38a9f50d23dd` | jobs-union |
| gms_61_1.json | gms | 61 | 1 | `0d250dc9-64c4-45ae-8bc2-fc0a9cdb5578` | jobs-union |
| gms_72_1.json | gms | 72 | 1 | `48d415ca-59de-4953-9aed-0c4156a09bc9` | jobs-union |
| gms_79_1.json | gms | 79 | 1 | `92adbe47-5ada-4f3b-8224-f58c80a4a2d5` | jobs-union |
| gms_83_1.json | gms | 83 | 1 | `ec876921-c363-4cc6-9c51-5bb8d57f9553` | jobs-union |
| gms_84_1.json | gms | 84 | 1 | `4936dff2-7121-4f46-b9eb-1ae541f4a85f` | jobs-union |
| gms_87_1.json | gms | 87 | 1 | `86da65d2-b9fa-4176-985a-6a5df586220c` | jobs-union |
| gms_92_1.json | gms | 92 | 1 | `db1dbfb3-4345-4731-9223-c40b0c7f6457` | jobs-union |
| gms_95_1.json | gms | 95 | 1 | `c794c706-aea3-4882-90a6-a3b7ee314f52` | jobs-union |
| jms_185_1.json | jms | 185 | 1 | `abedf3b4-1d7c-4b3b-bc52-70f62ab09418` | jobs-union |

## gms_12 policy (explicit user decision)

**gms_12 has no live atlas-data baseline tenant in this environment**
(confirmed independently by task-187 Task 1: absent from the `GET
/api/tenants` listing, and a direct `wget` against a synthesized tenant id
fails). Per an explicit user decision recorded in the Task 3 brief, `gms_12`
is **not drained** and **not fabricated**. Instead, `gms_12_1.json` mirrors
`gms_48_1.json` verbatim (identical `skills`/`jobs` arrays, and therefore
the identical `hash` value -- `hash` is a function of the id-sets only, not
of region/major/minor) with only the `region`/`major`/`minor` header fields
changed to `gms`/`12`/`1`.

This is a **documented approximation, not a live drain**: v0.12 predates
Pirate's release (Pirate shipped v0.62 per the meymink patch log, cited in
Task 1's audit) and its GM/SuperGM job ids at v0.12 are release-anchor-
grounded to sit at 500/510 (the same binding v0.48 uses; v0.48 is the
nearest provisioned pre-Pirate version, so mirroring it is the best
available stand-in for a v0.12 id-set). The stable id-set is an
approximation grounded in release-anchor evidence, not a live-baseline
measurement -- any downstream consumer treating `gms_12_1.json` as an
independently-verified snapshot would be wrong; it inherits gms_48's
verification status, nothing more.

## Raw job-list drains (not committed)

The raw `GET /api/data/jobs?page[size]=200` JSON:API responses for all 10
tenants were saved to a scratch directory during this drain for the
intermediate skill/job-set extraction, and are reproducible on demand from
the same tenants/headers documented above -- they are not stashed in this
repo (per Task 1's precedent of recording provenance rather than raw drain
dumps).
