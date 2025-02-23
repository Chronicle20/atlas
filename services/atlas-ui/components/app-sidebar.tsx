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

const tenants = [
    {
        id: "1e800451-586c-40cb-94c1-e277f97e7c2c",
        attributes: {
            region: "GMS",
            majorVersion: 12,
            minorVersion: 1,
        },
    },
    {
        id: "a3fe2199-1bd2-40f7-ba6a-d89d712848b5",
        attributes: {
            region: "GMS",
            majorVersion: 83,
            minorVersion: 1,
        },
    },
    {
        id: "74e95941-84d3-4e12-bf18-67adafb36ba6",
        attributes: {
            region: "GMS",
            majorVersion: 87,
            minorVersion: 1,
        },
    },
    {
        id: "03b7429a-3d62-4f4c-b511-677772853424",
        attributes: {
            region: "JMS",
            majorVersion: 185,
            minorVersion: 1,
        },
    },
];

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
                <TenantSwitcher tenants={tenants} />
            </SidebarHeader>
            <SidebarContent>
                <SidebarGroup>
                    <SidebarGroupContent>
                        <SidebarMenu>
                            {items.map((item) => (
                                <Collapsible>
                                <SidebarMenuItem key={item.title} className="group/collapsible">
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
                                                <SidebarMenuSubButton>
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
