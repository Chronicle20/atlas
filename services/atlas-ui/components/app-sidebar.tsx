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
import {Cog, MonitorCog, Shield} from "lucide-react";
import {Collapsible, CollapsibleContent, CollapsibleTrigger} from "@/components/ui/collapsible";
import Link from "next/link";
import Image from "next/image";
import {usePathname} from "next/navigation";
import {TenantSwitcher} from "@/components/app-tenant-switcher";
import logoImage from "@/app/logo.png";

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
            },
            {
                title: "Quests",
                url: "/quests"
            },
            {
                title: "Monsters",
                url: "/monsters"
            },
            {
                title: "Items",
                url: "/items"
            },
            {
                title: "Maps",
                url: "/maps"
            },
            {
                title: "Reactors",
                url: "/reactors"
            },
            {
                title: "Gachapons",
                url: "/gachapons"
            },
        ],
    },
    {
        title: "Security",
        url: "#",
        icon: Shield,
        children: [
            {
                title: "Bans",
                url: "/bans"
            },
            {
                title: "Login History",
                url: "/login-history"
            },
        ],
    },
    {
        title: "Administration",
        url: "#",
        icon: MonitorCog,
        children: [
            {
                title: "Bootstrap",
                url: "/setup"
            },
            {
                title: "Services",
                url: "/services"
            },
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
    const pathname = usePathname()

    return (
        <Sidebar>
            <SidebarHeader>
                <Link key="/" href="/">
                <div className="h-[210px] flex items-center justify-center">
                    <Image
                        src={logoImage}
                        alt="Logo"
                        width={210}
                        height={210}
                        priority
                    />
                </div>
                </Link>
                <TenantSwitcher />
            </SidebarHeader>
            <SidebarContent>
                <SidebarGroup>
                    <SidebarGroupContent>
                        <SidebarMenu>
                            {items.map((item) => {
                                const isGroupActive = item.children.some((child) =>
                                    pathname === child.url || pathname.startsWith(child.url + "/")
                                )
                                return (
                                <Collapsible key={item.title} defaultOpen={isGroupActive}>
                                <SidebarMenuItem className="group/collapsible">
                                    <CollapsibleTrigger asChild>
                                    <SidebarMenuButton>
                                        <item.icon />
                                        <span>{item.title}</span>
                                    </SidebarMenuButton>
                                    </CollapsibleTrigger>
                                    <CollapsibleContent>
                                    <SidebarMenuSub>
                                        {item.children.map((child) => {
                                            const isActive = pathname === child.url || pathname.startsWith(child.url + "/")
                                            return (
                                            <SidebarMenuSubItem key={child.title}>
                                                <SidebarMenuSubButton asChild isActive={isActive}>
                                                    <Link href={child.url}>
                                                        <span>{child.title}</span>
                                                    </Link>
                                                </SidebarMenuSubButton>
                                            </SidebarMenuSubItem>
                                            )
                                        })}
                                    </SidebarMenuSub>
                                    </CollapsibleContent>
                                </SidebarMenuItem>
                                </Collapsible>
                                )
                            })}
                        </SidebarMenu>
                    </SidebarGroupContent>
                </SidebarGroup>
            </SidebarContent>
            <SidebarFooter/>
        </Sidebar>
    )
}
