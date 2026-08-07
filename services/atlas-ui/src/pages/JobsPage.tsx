import { useEffect, useMemo } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { Briefcase } from "lucide-react";
import { useTenant } from "@/context/tenant-context";
import { useJobs } from "@/lib/hooks/api/useJobs";
import { useJobSkills } from "@/lib/hooks/api/useJobSkills";
import { useJobSkillDefinitions } from "@/lib/hooks/api/useJobSkillDefinitions";
import { useMediaQuery } from "@/hooks/use-media-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { JOB_GRAPH } from "@/lib/jobs/job-advancement-tree";
import {
  branchEntryOf,
  visibleRailGroups,
} from "@/components/features/jobs/rail-groups";
import { BranchRail } from "@/components/features/jobs/branch-rail";
import { AdvancementFlow } from "@/components/features/jobs/advancement-flow";
import {
  SkillList,
  type SkillListState,
} from "@/components/features/jobs/skill-list";
import { SkillDetail } from "@/components/features/jobs/skill-detail";
import { cn } from "@/lib/utils";

export function JobsPage() {
  const { jobId: jobIdParam } = useParams<{ jobId: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const { activeTenant } = useTenant();
  const isWide = useMediaQuery("(min-width: 1150px)");

  const jobsQuery = useJobs(activeTenant);
  // The tenant's actual job set, from GET /api/data/jobs. Empty while the query
  // is pending — which is why every consumer below is gated on isSuccess rather
  // than on the set being non-empty. TenantProvider calls queryClient.clear()
  // on every tenant switch, so "pending with an empty set" is the state right
  // after a switch, not an error.
  const available = useMemo<ReadonlySet<number>>(
    () => new Set((jobsQuery.data?.jobs ?? []).map((j) => Number(j.id))),
    [jobsQuery.data],
  );
  const groups = useMemo(
    () => (jobsQuery.isSuccess ? visibleRailGroups(available) : []),
    [jobsQuery.isSuccess, available],
  );
  const defaultJobId = groups[0]?.entries[0]?.id ?? 100;

  const parsedJobId = jobIdParam !== undefined ? Number(jobIdParam) : null;
  const jobIdValid =
    parsedJobId !== null &&
    Number.isInteger(parsedJobId) &&
    JOB_GRAPH[parsedJobId] !== undefined &&
    jobsQuery.isSuccess &&
    available.has(parsedJobId);
  // While the job set is pending the current jobId is retained: redirecting on
  // an unknown set would bounce a valid /jobs/112 to /jobs on every tenant
  // switch (design D10).
  const jobId = jobIdValid
    ? parsedJobId
    : jobsQuery.isSuccess
      ? defaultJobId
      : (parsedJobId ?? defaultJobId);

  // FR-1.2 / FR-7.3: an unknown or tenant-absent jobId normalizes to /jobs with
  // replace, so Back doesn't bounce. Gated on isSuccess so a pending or failed
  // job-set query never triggers it.
  useEffect(() => {
    if (
      activeTenant &&
      jobsQuery.isSuccess &&
      parsedJobId !== null &&
      !jobIdValid
    ) {
      navigate("/jobs", { replace: true });
    }
  }, [activeTenant, jobsQuery.isSuccess, parsedJobId, jobIdValid, navigate]);

  const entry = branchEntryOf(jobId);
  const jobName = JOB_GRAPH[jobId]?.name ?? `Job ${jobId}`;

  const skillsQuery = useJobSkills(activeTenant, jobId);
  const skillIds = useMemo(() => skillsQuery.data ?? [], [skillsQuery.data]);
  const {
    definitions,
    isLoading: defsLoading,
    isError: defsError,
  } = useJobSkillDefinitions(activeTenant, skillIds);

  const loading =
    jobsQuery.isPending ||
    skillsQuery.isLoading ||
    (skillIds.length > 0 && defsLoading);

  const skillParam = searchParams.get("skill");
  const selectedSkillId = skillParam !== null ? Number(skillParam) : null;
  const selectedDef = definitions.find((d) => d.id === selectedSkillId) ?? null;

  // D1: a ?skill= that never resolves for this job is stripped (replace) once
  // definitions settle.
  useEffect(() => {
    if (activeTenant && skillParam !== null && !loading && !selectedDef) {
      setSearchParams({}, { replace: true });
    }
  }, [activeTenant, skillParam, loading, selectedDef, setSearchParams]);

  const state: SkillListState = loading
    ? "loading"
    : skillsQuery.isError
      ? "error"
      : skillIds.length === 0
        ? "empty"
        : definitions.length === 0 && defsError
          ? "defs-failed"
          : "ready";

  const selectJob = (id: number) => navigate(`/jobs/${id}`); // push; drops ?skill=
  const selectSkill = (id: number) => setSearchParams({ skill: String(id) });
  const clearSkill = () => setSearchParams({});

  const detail = selectedDef ? (
    <SkillDetail key={selectedDef.id} def={selectedDef} accent={entry.accent} />
  ) : null;

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4 p-10 pb-6">
      <div className="flex flex-none items-center gap-2">
        <Briefcase className="h-6 w-6" />
        <h2 className="text-2xl font-bold tracking-tight">Jobs</h2>
      </div>

      {!activeTenant ? (
        <Card>
          <CardContent className="py-10 text-center text-muted-foreground">
            Select a tenant to browse its jobs and skills.
          </CardContent>
        </Card>
      ) : jobsQuery.isError ? (
        <Card>
          <CardContent
            data-testid="jobs-load-error"
            className="py-10 text-center text-muted-foreground"
          >
            Could not load this tenant&apos;s job list. Check that atlas-data is
            reachable and that this version has been ingested.
          </CardContent>
        </Card>
      ) : (
        <div
          className={cn(
            "grid min-h-0 flex-1 gap-3.5",
            isWide
              ? "grid-cols-[200px_minmax(340px,1fr)_minmax(480px,42%)]"
              : "grid-cols-[200px_minmax(0,1fr)]",
          )}
        >
          <BranchRail
            groups={groups}
            selectedEntryId={entry.id}
            onSelect={selectJob}
            isPending={jobsQuery.isPending}
          />

          <Card className="flex min-h-0 flex-col">
            <CardHeader className="pb-1">
              <CardTitle className="text-[15px]">Advancement</CardTitle>
            </CardHeader>
            <div className="flex-none border-b px-4 pb-3.5">
              <AdvancementFlow
                entryId={entry.id}
                available={available}
                selectedJobId={jobId}
                accent={entry.accent}
                onSelect={selectJob}
              />
            </div>
            <SkillList
              key={jobId}
              jobName={jobName}
              defs={definitions}
              state={state}
              selectedSkillId={selectedSkillId}
              accent={entry.accent}
              onSelect={selectSkill}
            />
          </Card>

          {isWide ? (
            <Card className="flex min-h-0 flex-col overflow-hidden">
              <CardHeader className="pb-1">
                <CardTitle className="text-[15px]">Skill Detail</CardTitle>
              </CardHeader>
              <div className="min-h-0 flex-1 overflow-y-auto">
                {detail ?? (
                  <div className="px-6 py-14 text-center text-muted-foreground">
                    Select a skill to inspect it
                  </div>
                )}
              </div>
            </Card>
          ) : (
            <Sheet
              open={selectedDef !== null}
              onOpenChange={(open) => {
                if (!open) clearSkill();
              }}
            >
              <SheetContent
                side="right"
                className="w-full overflow-y-auto sm:max-w-md"
              >
                <SheetHeader>
                  <SheetTitle className="sr-only">Skill detail</SheetTitle>
                </SheetHeader>
                {detail}
              </SheetContent>
            </Sheet>
          )}
        </div>
      )}
    </div>
  );
}
