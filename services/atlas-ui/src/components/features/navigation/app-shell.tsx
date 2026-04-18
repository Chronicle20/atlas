import { Outlet } from "react-router-dom";
import { SidebarProvider } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/app-sidebar";
import { ThemeToggle } from "@/components/theme-toggle";
import { SidebarToggle } from "@/components/sidebar-toggle";
import { BreadcrumbBar } from "@/components/features/navigation/BreadcrumbBar";
import { Separator } from "@/components/ui/separator";

export function AppShell() {
  return (
    <SidebarProvider>
      <AppSidebar />
      <main className="w-full flex h-screen flex-1 flex-col gap-2 pt-2">
        <header className="flex h-12 shrink-0 items-center gap-2 px-2">
          <SidebarToggle />
          <Separator orientation="vertical" className="mr-2 h-4" />
          <BreadcrumbBar
            maxItems={5}
            maxItemsMobile={2}
            showEllipsis={true}
            showLoadingStates={true}
          />
          <div className="ml-auto">
            <ThemeToggle />
          </div>
        </header>
        <div className="flex flex-1 flex-col overflow-hidden gap-4 p-2 pt-0">
          <div className="flex flex-1 flex-col overflow-hidden rounded-xl bg-sidebar">
            <Outlet />
          </div>
        </div>
      </main>
    </SidebarProvider>
  );
}
