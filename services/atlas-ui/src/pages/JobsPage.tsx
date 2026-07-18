import { useMemo } from "react";
import { Link } from "react-router-dom";
import { Briefcase, ChevronRight } from "lucide-react";
import { useTenant } from "@/context/tenant-context";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  JOB_GRAPH,
  visibleChildrenOf,
  visibleRoots,
} from "@/lib/jobs/job-advancement-tree";

function JobTreeNode({
  id,
  depth,
  major,
}: {
  id: number;
  depth: number;
  major: number;
}) {
  const entry = JOB_GRAPH[id];
  const name = entry?.name ?? `Job ${id}`;
  const children = visibleChildrenOf(id, major);
  const indent = { paddingLeft: depth * 16 } as const;

  if (children.length === 0) {
    return (
      <div style={indent} className="py-1">
        <Link
          to={`/jobs/${id}`}
          className="text-sm text-primary underline-offset-2 hover:underline"
        >
          {name}
        </Link>
      </div>
    );
  }

  return (
    <Collapsible defaultOpen={depth === 0}>
      <div style={indent} className="flex items-center gap-1 py-1">
        <CollapsibleTrigger
          aria-label={`Toggle ${name}`}
          className="group flex h-6 w-6 items-center justify-center rounded hover:bg-muted cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <ChevronRight className="h-4 w-4 transition-transform group-data-[state=open]:rotate-90" />
        </CollapsibleTrigger>
        <Link
          to={`/jobs/${id}`}
          className="text-sm font-medium text-primary underline-offset-2 hover:underline"
        >
          {name}
        </Link>
      </div>
      <CollapsibleContent>
        {children.map((childId) => (
          <JobTreeNode
            key={childId}
            id={childId}
            depth={depth + 1}
            major={major}
          />
        ))}
      </CollapsibleContent>
    </Collapsible>
  );
}

export function JobsPage() {
  const { activeTenant } = useTenant();

  const major = activeTenant?.attributes.majorVersion ?? 0;
  const roots = useMemo(
    () => (activeTenant ? visibleRoots(major) : []),
    [activeTenant, major],
  );

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 overflow-y-auto p-10 pb-16">
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
          <CardContent className="space-y-1">
            {roots.map((rootId) => (
              <JobTreeNode key={rootId} id={rootId} depth={0} major={major} />
            ))}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
