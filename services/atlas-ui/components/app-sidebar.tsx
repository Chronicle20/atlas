"use client"

import {
    Sidebar,
    SidebarContent,
    SidebarFooter,
    SidebarGroup,
    SidebarGroupContent,
    SidebarHeader,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    SidebarMenuSub,
    SidebarMenuSubButton,
    SidebarMenuSubItem,
} from "@/components/ui/sidebar"
import {ChartNoAxesCombined, Cog, MonitorCog} from "lucide-react";
import {Collapsible, CollapsibleContent, CollapsibleTrigger} from "@/components/ui/collapsible";
import Link from "next/link";
import {TenantSwitcher} from "@/components/app-tenant-switcher";

// Menu items.
const items = [
    {
        title: "Dashboards",
        url: "#",
        icon: ChartNoAxesCombined,
        children: [
            {
                title: "Overview",
                url: "/dashboard/overview"
            }
        ],
    },
    {
        title: "Operations",
        url: "#",
        icon: Cog,
        children: [
            {
                title: "Accounts",
                url: "/account"
            },
            {
                title: "Characters",
                url: "/character"
            },
            {
                title: "Drops",
                url: "/drop"
            }
        ],
    },
    {
        title: "Administration",
        url: "#",
        icon: MonitorCog,
        children: [
            {
                title: "Tenants",
                url: "/tenant"
            },
            {
                title: "Templates",
                url: "/template"
            },
        ],
    },
]

export function AppSidebar() {
    return (
        <Sidebar>
            <SidebarHeader>
                <Link key="/" href="/">
                <div className="h-[200px] flex items-center justify-center">
                    LOGO
                </div>
                </Link>
                <TenantSwitcher />
            </SidebarHeader>
            <SidebarContent>
                <SidebarGroup>
                    <SidebarGroupContent>
                        <SidebarMenu>
                            {items.map((item) => (
                                <Collapsible key={item.title}>
                                <SidebarMenuItem className="group/collapsible">
                                    <CollapsibleTrigger asChild>
                                    <SidebarMenuButton asChild>
                                        <a href={item.url}>
                                            <item.icon />
                                            <span>{item.title}</span>
                                        </a>
                                    </SidebarMenuButton>
                                    </CollapsibleTrigger>
                                    <CollapsibleContent>
                                    <SidebarMenuSub>
                                        {item.children.map((child) => (
                                            <SidebarMenuSubItem key={child.title}>
                                                <SidebarMenuSubButton asChild>
                                                    <a href={child.url}>
                                                        <span>{child.title}</span>
                                                    </a>
                                                </SidebarMenuSubButton>
                                            </SidebarMenuSubItem>
                                        ))}
                                    </SidebarMenuSub>
                                    </CollapsibleContent>
                                </SidebarMenuItem>
                                </Collapsible>
                            ))}
                        </SidebarMenu>
                    </SidebarGroupContent>
                </SidebarGroup>
            </SidebarContent>
            <SidebarFooter/>
        </Sidebar>
    )
}
