import { useMemo } from "react";
import { Link } from "react-router-dom";
import { Briefcase } from "lucide-react";
import { useTenant } from "@/context/tenant-context";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { JOB_HIERARCHY, filterHierarchy, jobNodeName } from "@/lib/jobs-hierarchy";

export function JobsPage() {
  const { activeTenant } = useTenant();

  const tree = useMemo(
    () => (activeTenant ? filterHierarchy(JOB_HIERARCHY, activeTenant.attributes.majorVersion) : []),
    [activeTenant],
  );

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
      <div className="flex items-center gap-2">
        <Briefcase className="h-6 w-6" />
        <h2 className="text-2xl font-bold tracking-tight">Jobs</h2>
      </div>

      {!activeTenant ? (
        <Card>
          <CardContent className="py-10 text-center text-muted-foreground">
            Select a tenant to browse its jobs and skills.
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>Job Hierarchy</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {tree.map((archetype) => (
              <Collapsible key={archetype.name} defaultOpen>
                <CollapsibleTrigger className="text-lg font-semibold py-1">
                  {archetype.name}
                </CollapsibleTrigger>
                <CollapsibleContent className="pl-4 space-y-1">
                  {archetype.classes.map((cls) => (
                    <Collapsible key={cls.name}>
                      <CollapsibleTrigger className="font-medium py-1">{cls.name}</CollapsibleTrigger>
                      <CollapsibleContent className="pl-4 flex flex-wrap gap-2 py-1">
                        {cls.jobs.map((job) => (
                          <Link
                            key={job.jobId}
                            to={`/jobs/${job.jobId}`}
                            className="text-sm text-primary underline-offset-2 hover:underline"
                          >
                            {jobNodeName(job)}
                          </Link>
                        ))}
                      </CollapsibleContent>
                    </Collapsible>
                  ))}
                </CollapsibleContent>
              </Collapsible>
            ))}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
