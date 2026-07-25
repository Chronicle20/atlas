import { Cog, MonitorCog, Shield, Wrench, type LucideIcon } from "lucide-react";

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
