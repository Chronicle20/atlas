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
import {Cog, MonitorCog} from "lucide-react";
import {Collapsible, CollapsibleContent, CollapsibleTrigger} from "@/components/ui/collapsible";
import Link from "next/link";
import {TenantSwitcher} from "@/components/app-tenant-switcher";

// Menu items.
const items = [
    {
        title: "Operations",
        url: "#",
        icon: Cog,
        children: [
            {
                title: "Accounts",
                url: "/accounts"
            },
            {
                title: "Characters",
                url: "/characters"
            },
            {
                title: "Guilds",
                url: "/guilds"
            },
            {
                title: "NPCs",
                url: "/npcs"
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
                url: "/tenants"
            },
            {
                title: "Templates",
                url: "/templates"
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
                                                    <Link href={child.url}>
                                                        <span>{child.title}</span>
                                                    </Link>
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
