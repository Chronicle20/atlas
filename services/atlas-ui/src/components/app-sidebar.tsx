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
    SidebarSeparator,
} from "@/components/ui/sidebar"
import {Fragment} from "react";
import {Collapsible, CollapsibleContent, CollapsibleTrigger} from "@/components/ui/collapsible";
import { Link } from "react-router-dom";
import { useLocation } from "react-router-dom";
import {TenantSwitcher} from "@/components/app-tenant-switcher";
import { sidebarItems } from "@/components/app-sidebar-items";
const logoImage = "/logo.png";

export function AppSidebar() {
    const pathname = useLocation().pathname

    return (
        <Sidebar>
            <SidebarHeader>
                <Link key="/" to="/">
                <div className="h-[210px] flex items-center justify-center">
                    <img
                        src={logoImage}
                        alt="Logo"
                        width={210}
                        height={210}
                    />
                </div>
                </Link>
                <TenantSwitcher />
            </SidebarHeader>
            <SidebarContent>
                <SidebarGroup>
                    <SidebarGroupContent>
                        <SidebarMenu>
                            {sidebarItems.map((item) => {
                                const isGroupActive = item.children.some((child) =>
                                    pathname === child.url || pathname.startsWith(child.url + "/")
                                )
                                return (
                                <Fragment key={item.title}>
                                {item.separated && <SidebarSeparator />}
                                <Collapsible defaultOpen={isGroupActive}>
                                <SidebarMenuItem className="group/collapsible">
                                    <CollapsibleTrigger asChild>
                                    <SidebarMenuButton className={item.caption ? "h-auto" : undefined}>
                                        <item.icon />
                                        <div className="grid flex-1 text-left leading-tight">
                                            <span>{item.title}</span>
                                            {item.caption && (
                                                <span className="text-xs text-muted-foreground">{item.caption}</span>
                                            )}
                                        </div>
                                    </SidebarMenuButton>
                                    </CollapsibleTrigger>
                                    <CollapsibleContent>
                                    <SidebarMenuSub>
                                        {item.children.map((child) => {
                                            const isActive = pathname === child.url || pathname.startsWith(child.url + "/")
                                            return (
                                            <SidebarMenuSubItem key={child.title}>
                                                <SidebarMenuSubButton asChild isActive={isActive}>
                                                    <Link to={child.url}>
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
                                </Fragment>
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
