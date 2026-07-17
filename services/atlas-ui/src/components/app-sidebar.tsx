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
} from "@/components/ui/sidebar";
import { Cog, MonitorCog, Shield, Wrench, type LucideIcon } from "lucide-react";
import { Fragment } from "react";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Link } from "react-router-dom";
import { useLocation } from "react-router-dom";
import { TenantSwitcher } from "@/components/app-tenant-switcher";
const logoImage = "/logo.png";

interface SidebarChildItem {
  title: string;
  url: string;
}

export interface SidebarGroupItem {
  title: string;
  url: string;
  icon: LucideIcon;
  /** Render a separator above this group (Deployment only). */
  separated?: boolean;
  /** Muted caption under the group label (Deployment only). */
  caption?: string;
  children: SidebarChildItem[];
}

// Menu items, grouped by blast radius: everything outside Deployment follows
// the tenant switcher; nothing inside it does. Exported so the sync test can
// assert Deployment children agree with isDeploymentRoute.
export const sidebarItems: SidebarGroupItem[] = [
  {
    title: "Operations",
    url: "#",
    icon: Cog,
    children: [
      { title: "Accounts", url: "/accounts" },
      { title: "Characters", url: "/characters" },
      { title: "Guilds", url: "/guilds" },
      { title: "NPCs", url: "/npcs" },
      { title: "Quests", url: "/quests" },
      { title: "Monsters", url: "/monsters" },
      { title: "Items", url: "/items" },
      { title: "Jobs", url: "/jobs" },
      { title: "Merchants", url: "/merchants" },
      { title: "Marketplace", url: "/marketplace" },
      { title: "Maps", url: "/maps" },
      { title: "Reactors", url: "/reactors" },
      { title: "Reward Pools", url: "/reward-pools" },
    ],
  },
  {
    title: "Security",
    url: "#",
    icon: Shield,
    children: [
      { title: "Bans", url: "/bans" },
      { title: "Login History", url: "/login-history" },
    ],
  },
  {
    title: "Setup",
    url: "#",
    icon: Wrench,
    children: [{ title: "Setup", url: "/setup" }],
  },
  {
    title: "Deployment",
    url: "#",
    icon: MonitorCog,
    separated: true,
    caption: "Applies to all tenants",
    children: [
      { title: "Templates", url: "/templates" },
      { title: "Tenants", url: "/tenants" },
      { title: "Services", url: "/services" },
      { title: "Baselines", url: "/baselines" },
    ],
  },
];

export function AppSidebar() {
  const pathname = useLocation().pathname;

  return (
    <Sidebar>
      <SidebarHeader>
        <Link key="/" to="/">
          <div className="h-[210px] flex items-center justify-center">
            <img src={logoImage} alt="Logo" width={210} height={210} />
          </div>
        </Link>
        <TenantSwitcher />
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              {sidebarItems.map((item) => {
                const isGroupActive = item.children.some(
                  (child) =>
                    pathname === child.url ||
                    pathname.startsWith(child.url + "/"),
                );
                return (
                  <Fragment key={item.title}>
                    {item.separated && <SidebarSeparator />}
                    <Collapsible defaultOpen={isGroupActive}>
                      <SidebarMenuItem className="group/collapsible">
                        <CollapsibleTrigger asChild>
                          <SidebarMenuButton
                            className={item.caption ? "h-auto" : undefined}
                          >
                            <item.icon />
                            <div className="grid flex-1 text-left leading-tight">
                              <span>{item.title}</span>
                              {item.caption && (
                                <span className="text-xs text-muted-foreground">
                                  {item.caption}
                                </span>
                              )}
                            </div>
                          </SidebarMenuButton>
                        </CollapsibleTrigger>
                        <CollapsibleContent>
                          <SidebarMenuSub>
                            {item.children.map((child) => {
                              const isActive =
                                pathname === child.url ||
                                pathname.startsWith(child.url + "/");
                              return (
                                <SidebarMenuSubItem key={child.title}>
                                  <SidebarMenuSubButton
                                    asChild
                                    isActive={isActive}
                                  >
                                    <Link to={child.url}>
                                      <span>{child.title}</span>
                                    </Link>
                                  </SidebarMenuSubButton>
                                </SidebarMenuSubItem>
                              );
                            })}
                          </SidebarMenuSub>
                        </CollapsibleContent>
                      </SidebarMenuItem>
                    </Collapsible>
                  </Fragment>
                );
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter />
    </Sidebar>
  );
}
