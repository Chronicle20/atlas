"use client"

import {useParams} from "next/navigation";
import {useTenant} from "@/context/tenant-context";
import {Separator} from "@/components/ui/separator";
import {DetailSidebar} from "@/components/detail-sidebar";
import {ScrollArea} from "@/components/ui/scroll-area";

interface TenantDetailLayoutProps {
    children: React.ReactNode
}

export default function TenantDetailLayout({ children }: TenantDetailLayoutProps) {
    const { id } = useParams(); // Get tenants ID from URL
    const {tenants} = useTenant()
    const tenant = tenants.find((t) => t.id === id);


    const sidebarNavItems = [
        {
            title: "Global Properties",
            href: "/tenants/" + id + "/properties",
        },
        {
            title: "Character Templates",
            href: "/tenants/" + id + "/character/templates",
        },
        {
            title: "Socket Handlers",
            href: "/tenants/" + id + "/handlers",
        },
        {
            title: "Socket Writers",
            href: "/tenants/" + id + "/writers",
        },
        {
            title: "Worlds",
            href: "/tenants/" + id + "/worlds",
        },
    ]

    return (
        <>
            <div className="flex flex-1 flex-col overflow-hidden space-y-6 p-10 pb-16">
                <div className="space-y-0.5">
                    <h2 className="text-2xl font-bold tracking-tight">Tenant Details</h2>
                    <p className="text-muted-foreground">
                        {tenant?.id}
                    </p>
                </div>
                <Separator className="my-6" />
                <div className="flex flex-1 flex-col overflow-hidden space-y-8 lg:flex-row lg:space-x-12 lg:space-y-0">
                    <aside className="lg:w-1/7">
                        <DetailSidebar items={sidebarNavItems} />
                    </aside>
                    <ScrollArea className="flex flex-1 flex-col">
                        <div className="flex-1 lg:max-w-4xl">{children}</div>
                    </ScrollArea>
                </div>
            </div>
        </>
    )
}