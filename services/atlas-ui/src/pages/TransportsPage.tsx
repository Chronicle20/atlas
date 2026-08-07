import { Suspense, useMemo } from "react";
import { useSearchParams } from "react-router-dom";
import { Ship } from "lucide-react";

import { DataTable } from "@/components/data-table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useTenant } from "@/context/tenant-context";
import { useScheduledRoutes, useVessels } from "@/lib/hooks/api/useTransports";
import { FreshnessIndicator } from "@/components/features/transports/FreshnessIndicator";
import { compareRoutesBySeverityThenName } from "@/components/features/transports/transport-format";
import { createScheduledRouteColumns } from "@/pages/transports-columns";

const TABS = ["scheduled", "instance", "vessels"] as const;
type TransportTab = (typeof TABS)[number];

function isTransportTab(value: string | null): value is TransportTab {
  return value !== null && (TABS as readonly string[]).includes(value);
}

export function TransportsPage() {
  return (
    <Suspense>
      <TransportsPageContent />
    </Suspense>
  );
}

function TransportsPageContent() {
  const { activeTenant } = useTenant();
  const [searchParams, setSearchParams] = useSearchParams();

  const requestedTab = searchParams.get("tab");
  const activeTab: TransportTab = isTransportTab(requestedTab)
    ? requestedTab
    : "scheduled";

  const scheduledQuery = useScheduledRoutes();
  const vesselsQuery = useVessels();

  const routes = useMemo(
    () =>
      [...(scheduledQuery.data ?? [])].sort(compareRoutesBySeverityThenName),
    [scheduledQuery.data],
  );
  const vessels = useMemo(() => vesselsQuery.data ?? [], [vesselsQuery.data]);

  const scheduledColumns = useMemo(
    () => createScheduledRouteColumns({ tenant: activeTenant, vessels }),
    [activeTenant, vessels],
  );

  const handleTabChange = (value: string) => {
    const next = new URLSearchParams(searchParams);
    if (value === "scheduled") {
      next.delete("tab");
    } else {
      next.set("tab", value);
    }
    setSearchParams(next, { replace: true });
  };

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-2">
          <Ship className="h-6 w-6" />
          <h2 className="text-2xl font-bold tracking-tight">Transports</h2>
        </div>
        <FreshnessIndicator
          dataUpdatedAt={scheduledQuery.dataUpdatedAt}
          isFetching={scheduledQuery.isFetching}
          isError={scheduledQuery.isError}
        />
      </div>

      <Tabs
        value={activeTab}
        onValueChange={handleTabChange}
        className="flex-1 min-h-0 flex flex-col"
      >
        <TabsList>
          <TabsTrigger value="scheduled">Scheduled {routes.length}</TabsTrigger>
          <TabsTrigger value="instance">Instance</TabsTrigger>
          <TabsTrigger value="vessels">Vessels {vessels.length}</TabsTrigger>
        </TabsList>

        <TabsContent
          value="scheduled"
          className="flex-1 min-h-0 overflow-x-auto"
        >
          <DataTable
            columns={scheduledColumns}
            data={routes}
            onRefresh={() => void scheduledQuery.refetch()}
            isRefreshing={scheduledQuery.isFetching}
          />
        </TabsContent>

        <TabsContent
          value="instance"
          className="flex-1 min-h-0 overflow-x-auto"
        />

        <TabsContent
          value="vessels"
          className="flex-1 min-h-0 overflow-x-auto"
        />
      </Tabs>
    </div>
  );
}
