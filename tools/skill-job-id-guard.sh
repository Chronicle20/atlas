#!/bin/sh
# skill-job-id-guard.sh — bans raw comparisons against the canonical
# job/skill "…Id" wire constants that task-187 identified as VERSION-
# DIVERGENT (their numeric wire value means something different at
# different client versions — e.g. job.GmId (500) is Gm at v0.48 but
# Pirate at v0.61+; skill.SuperGmHideId (5101004) is SuperGmHide at v0.48
# but BrawlerCorkscrewBlow at v0.61+). A raw `jobId == job.SuperGmId` (or
# `job.IsA(jobId, job.SuperGmId)`) compare silently breaks at whichever
# version the const's wire value doesn't hold for real code has to resolve
# through the version-aware SkillJobSet (constants.For(...).Job.Resolve /
# .Skill.Resolve) and branch on the resulting Identity instead.
#
# The banned const list is DERIVED from
# docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv so it
# grows automatically as future audit passes add divergent ids (OQ-6) --
# nothing here is hand-maintained. Two filters apply, per design.md §8.3 /
# plan.md's "Version-scope discipline" note (only GM/SuperGM jobs
# 500/510/900/910, Pirate/Brawler, skill 5101004, and the Big Bang reorg set
# remap -- version-stable roots like Cygnus/Aran/Evan/DualBlade "do not shift
# across the provisioned GMS range and stay Id-keyed with an inline audit
# citation"):
#
#   1. identityName must be a bare Go identifier (drops the Big Bang
#      merge/rename/split documentation rows, e.g. "HPBoost [merge-target,
#      ...]", and the annotated "DualBlade (unreleased WZ stub)" row).
#   2. The row must describe a TRUE divergence, not just CSV presence: after
#      normalizing each identityName to its leading bare-identifier prefix
#      (so "DualBlade (unreleased WZ stub)" and "DualBlade" compare equal),
#      a (region,domain,wireId) is divergent iff either (a) the same wireId
#      is bound to more than one distinct identity across the CSV (the
#      Gm<->Pirate / SuperGm<->Brawler collision), or (b) the same identity
#      is bound to more than one distinct wireId across the CSV (Gm=500@v48
#      but Gm=900@v61+; SuperGm=510@v48 but SuperGm=910@v61+). This is what
#      correctly separates the true GM/SuperGM/Pirate/Brawler divergent set
#      from CSV rows that merely document a pre-release WZ-name/identity
#      normalization for an otherwise version-stable root -- e.g. Evan
#      (wireId 2001, only ever Evan), Legend/Aran (wireId 2000, only ever
#      Legend), Noblesse/Cygnus (wireId 1000, only ever Noblesse), and
#      DualBlade (wireId 430, only ever DualBlade) each have exactly one
#      identity at exactly one wireId in the CSV and are correctly excluded
#      -- confirmed against plan.md's explicit "Version-stable roots ...
#      Cygnus 1xxx, Aran 20xx, Evan 22xx ... do not shift" and the matching
#      "version-stable per task-187 audit" comments already in
#      character_attack_common.go / hp_mp_gain.go / processor.go. A naive
#      "any bare-identifier CSV row" filter would ban job.EvanId and trip on
#      6 legitimate, already-correct `case job.EvanId:` sites in
#      atlas-cashshop/atlas-channel/atlas-consumables/atlas-npc-shops/
#      atlas-pets/atlas-query-aggregator's HasSPTable() -- those wire ids
#      never collide with anything at any provisioned version, so a raw
#      compare is not a bug.
#
# Excludes: libs/atlas-constants/** (the resolver + generated consts
# legitimately reference these names), *_gen.go, *_test.go, and any line
# whose // comment was stripped before matching (the migrated sites document
# the divergence with an explanatory comment that legitimately names the
# raw const -- e.g. "a raw job.IsA(c.JobId(), job.SuperGmId) compare would
# break at v48" -- these must NOT trip the guard). A line may also carry a
# `//skill-job-id-guard:allow: <reason>` trailing annotation for a narrow,
# justified exception; the guard honors it.
#
# Scope note: this matches NAMED-const usage only (`<alias>.<Name>Id`, any
# import alias) -- it does NOT scan for raw numeric-literal wire-id
# comparisons (`== 500`, `== job.Id(500)`). The codebase convention is 100%
# named-const for these ids (confirmed: zero raw-literal wire-id compares
# found), and a bare-literal scan for numbers like 500/900 would be
# prohibitively noisy against unrelated numeric literals repo-wide.
#
# The wireId/identity collision maps above are built across ALL rows
# regardless of `major`/`minor` (not scoped per-version) -- harmless today
# (fails safe toward over-banning, never under-banning) but worth knowing if
# a future audit pass adds rows where that matters.
#
# Only services/**/*.go and libs/**/*.go (minus atlas-constants) are
# scanned. Run from the repo root; non-empty diagnostics -> non-zero exit.
set -eu

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CSV="$ROOT/docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv"

if [ ! -f "$CSV" ]; then
    echo "skill-job-id-guard: divergences.csv not found at $CSV" >&2
    exit 1
fi

python3 - "$ROOT" "$CSV" <<'PY'
import csv
import os
import re
import sys

root, csv_path = sys.argv[1], sys.argv[2]

# ---- Step 1: derive the banned const list from divergences.csv --------
bare_identifier = re.compile(r'^[A-Za-z][A-Za-z0-9]*$')
leading_identifier = re.compile(r'^[A-Za-z][A-Za-z0-9]*')

rows = []  # (domain, wireId, rawName, normalizedName)
with open(csv_path, newline='', encoding='utf-8') as f:
    for row in csv.DictReader(f):
        domain = row.get('domain', '')
        wire_id = row.get('wireId', '')
        name = row.get('identityName', '')
        if domain not in ('job', 'skill'):
            continue
        m = leading_identifier.match(name)
        if not m:
            continue
        rows.append((domain, wire_id, name, m.group(0)))

# Build the two collision maps (per domain, since job/skill wireIds are
# independent numbering spaces): wireId -> {normalized identities} and
# normalized identity -> {wireIds}.
wire_to_identities = {}
identity_to_wires = {}
for domain, wire_id, _raw, norm in rows:
    wire_to_identities.setdefault((domain, wire_id), set()).add(norm)
    identity_to_wires.setdefault((domain, norm), set()).add(wire_id)

def is_true_divergent(domain, wire_id, norm):
    return (len(wire_to_identities[(domain, wire_id)]) > 1
            or len(identity_to_wires[(domain, norm)]) > 1)

# job wireIds confirmed truly divergent (collide with another identity, or
# the same identity's wire value shifts across versions) -- currently
# {500, 510, 900, 910} (Gm/SuperGm/Pirate/Brawler), derived, not hardcoded.
divergent_job_wire_ids = set()
for domain, wire_id, _raw, norm in rows:
    if domain == 'job' and is_true_divergent(domain, wire_id, norm):
        divergent_job_wire_ids.add(wire_id)

def skill_job_prefix(wire_id):
    # MapleStory skill ids are jobId*10000 + suffix (e.g. 5101004 = job 510
    # + suffix 1004). A skill bound under an already-divergent job inherits
    # the divergence even where the CSV only documents the job-level
    # collision (e.g. GmHaste/5001000 -- the CSV cites the Pirate skill
    # names for job 500 as evidence text inside the JOB row, not as its own
    # skill-domain row).
    try:
        return str(int(wire_id) // 10000)
    except ValueError:
        return None

const_names = set()
for domain, wire_id, raw, norm in rows:
    if not bare_identifier.match(raw):
        # Drops bracketed Big Bang merge/rename/split documentation rows
        # and the annotated "(unreleased WZ stub)" row from the OUTPUT
        # (they still counted toward the collision maps above via their
        # normalized prefix).
        continue
    if domain == 'job':
        divergent = is_true_divergent(domain, wire_id, norm)
    else:  # skill
        divergent = (is_true_divergent(domain, wire_id, norm)
                     or skill_job_prefix(wire_id) in divergent_job_wire_ids)
    if not divergent:
        # Single identity, single wireId, not bound under a divergent job --
        # a version-stable root documented for WZ-name/availability reasons,
        # not a true remap. (Evan/Legend/Noblesse/DualBlade land here.)
        continue
    const_names.add(raw + 'Id')

if not const_names:
    print("skill-job-id-guard: derived an EMPTY banned-const list from "
          "divergences.csv -- refusing to run a no-op guard", file=sys.stderr)
    sys.exit(1)

# Match <anything>.<ConstName> so any import alias (job, job2, skill3, ...)
# is caught, per-const, word-bounded on both ends.
const_pattern = re.compile(
    r'\b[A-Za-z_][A-Za-z0-9_]*\.(' + '|'.join(sorted(re.escape(c) for c in const_names)) + r')\b'
)

# Comparison/predicate context: ==, !=, `case `, Is(, IsA(.
context_pattern = re.compile(r'==|!=|\bcase\b|\bIs\(|\bIsA\(')

allow_pattern = re.compile(r'//\s*skill-job-id-guard:allow\b')

def strip_line_comment(line):
    idx = line.find('//')
    if idx == -1:
        return line
    return line[:idx]

# ---- Step 2: walk services/**/*.go and libs/**/*.go (minus atlas-constants) ----
scan_roots = [os.path.join(root, 'services'), os.path.join(root, 'libs')]
excluded_dir = os.path.join(root, 'libs', 'atlas-constants')

hits = []
for base in scan_roots:
    for dirpath, dirnames, filenames in os.walk(base):
        if dirpath == excluded_dir or dirpath.startswith(excluded_dir + os.sep):
            dirnames[:] = []
            continue
        for fname in filenames:
            if not fname.endswith('.go'):
                continue
            if fname.endswith('_gen.go') or fname.endswith('_test.go'):
                continue
            path = os.path.join(dirpath, fname)
            try:
                with open(path, encoding='utf-8') as fh:
                    lines = fh.readlines()
            except (UnicodeDecodeError, OSError):
                continue
            for lineno, raw_line in enumerate(lines, start=1):
                line = raw_line.rstrip('\n')
                if allow_pattern.search(line):
                    continue
                stripped = strip_line_comment(line)
                if not const_pattern.search(stripped):
                    continue
                if not context_pattern.search(stripped):
                    continue
                rel = os.path.relpath(path, root)
                hits.append((rel, lineno, line.strip()))

if hits:
    for rel, lineno, text in sorted(hits):
        print("%s:%d: %s" % (rel, lineno, text))
    print("")
    print("skill-job-id-guard: FAIL — %d raw comparison(s) against a "
          "version-divergent job/skill id const found. Resolve the wire id "
          "to its version-aware Identity via constants.For(region, major, "
          "minor).Job.Resolve/.Skill.Resolve and compare Identities instead "
          "(see services/atlas-channel/.../skill/handler/version_resolve.go)."
          % len(hits))
    sys.exit(1)

print("skill-job-id-guard: clean (%d divergent const(s) checked)" % len(const_names))
PY
