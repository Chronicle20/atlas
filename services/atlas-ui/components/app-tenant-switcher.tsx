"use client"

import * as React from "react"
import {ChevronsUpDown, Plus} from "lucide-react"

import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuShortcut,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    useSidebar,
} from "@/components/ui/sidebar"

type Tenant = {
    tenants: {
        id: string;
        attributes: {
            region: string;
        };
    }[]
};

export function TenantSwitcher({
                                   tenants,
                               }: { tenants: Tenant[] }) {
    const {isMobile} = useSidebar()
    const [activeTenant, setActiveTenant] = React.useState(tenants[0])

    return (
        <SidebarMenu>
            <SidebarMenuItem>
                <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                        <SidebarMenuButton
                            size="lg"
                            className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
                        >
                            <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-semibold">
                  {activeTenant.id}
                </span>
                                <span className="truncate text-xs">{activeTenant.attributes.region} - {activeTenant.attributes.majorVersion}.{activeTenant.attributes.minorVersion}</span>
                            </div>
                            <ChevronsUpDown className="ml-auto"/>
                        </SidebarMenuButton>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent
                        className="w-[--radix-dropdown-menu-trigger-width] min-w-56 rounded-lg"
                        align="start"
                        side={isMobile ? "bottom" : "right"}
                        sideOffset={4}
                    >
                        <DropdownMenuLabel className="text-xs text-muted-foreground">
                            Tenants
                        </DropdownMenuLabel>
                        {tenants.map((tenant, index) => (
                            <DropdownMenuItem
                                key={tenant.id}
                                onClick={() => setActiveTenant(tenant)}
                                className="gap-2 p-2"
                            >
                                {tenant.id}
                            </DropdownMenuItem>
                        ))}
                        <DropdownMenuSeparator/>
                        <DropdownMenuItem className="gap-2 p-2">
                            <div className="flex size-6 items-center justify-center rounded-md border bg-background">
                                <Plus className="size-4"/>
                            </div>
                            <div className="font-medium text-muted-foreground">Add tenant</div>
                        </DropdownMenuItem>
                    </DropdownMenuContent>
                </DropdownMenu>
            </SidebarMenuItem>
        </SidebarMenu>
    )
}