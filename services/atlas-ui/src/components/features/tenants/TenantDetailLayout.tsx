import { type ReactNode } from "react";
import { useParams } from "react-router-dom";
import { Separator } from "@/components/ui/separator";
import { DetailSidebar } from "@/components/detail-sidebar";
import {
  DetailActionBar,
  DetailActionBarProvider,
} from "@/components/DetailActionBarContext";
import { ConfigExportButton } from "@/components/features/config/ConfigExportButton";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { supportsMapleLife } from "@/components/features/characters/maple-life/mapleLifeSupport";

interface TenantDetailLayoutProps {
  children: ReactNode;
}

export function TenantDetailLayout({ children }: TenantDetailLayoutProps) {
  const { id } = useParams();
  // Shares the React Query cache with TenantsMapleLifePage, so this adds no
  // extra request beyond what the page already issues.
  const tenantQuery = useTenantConfiguration(id ?? "");
  const sidebarNavItems = [
    { title: "Global Properties", href: `/tenants/${id}/properties` },
    {
      title: "Character Templates",
      href: `/tenants/${id}/character/templates`,
    },
    { title: "Character Presets", href: `/tenants/${id}/character/presets` },
    ...(supportsMapleLife(tenantQuery.data?.attributes.socket)
      ? [{ title: "Maple Life", href: `/tenants/${id}/character/maple-life` }]
      : []),
    { title: "Socket Handlers", href: `/tenants/${id}/handlers` },
    { title: "Socket Writers", href: `/tenants/${id}/writers` },
    { title: "Worlds", href: `/tenants/${id}/worlds` },
    { title: "MTS Configuration", href: `/tenants/${id}/mts-config` },
  ];
  return (
    <DetailActionBarProvider>
      <div className="flex flex-1 flex-col overflow-hidden space-y-6 p-10 pb-6">
        <div className="flex items-start justify-between gap-4">
          <div className="space-y-0.5">
            <h2 className="text-2xl font-bold tracking-tight">
              Tenant Details
            </h2>
            <p className="text-muted-foreground">{id}</p>
          </div>
          <ConfigExportButton kind="tenant" id={id} />
        </div>
        <Separator className="my-6" />
        <div className="flex flex-1 flex-col overflow-hidden space-y-8 lg:flex-row lg:space-x-12 lg:space-y-0">
          <aside className="lg:w-1/5">
            <DetailSidebar items={sidebarNavItems} />
          </aside>
          <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
            <div className="flex-1 overflow-y-auto px-2 py-1">{children}</div>
            <div className="px-2">
              <DetailActionBar />
            </div>
          </div>
        </div>
      </div>
    </DetailActionBarProvider>
  );
}
