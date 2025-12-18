import type {Metadata} from "next";
import "./globals.css";
import {SidebarProvider} from "@/components/ui/sidebar"
import {AppSidebar} from "@/components/app-sidebar"
import {ThemeProvider} from "next-themes";
import {ThemeToggle} from "@/components/theme-toggle";
import {SidebarToggle} from "@/components/sidebar-toggle";
import {TenantProvider} from "@/context/tenant-context";
import {Toaster} from "@/components/ui/sonner";
import {QueryProvider} from "@/components/providers/query-provider";
import {BreadcrumbBar} from "@/components/features/navigation/BreadcrumbBar";
import {Separator} from "@/components/ui/separator";

export const metadata: Metadata = {
    title: "AtlasMS",
};

export default function RootLayout({children,}: Readonly<{ children: React.ReactNode; }>) {
    return (
        <html lang="en" suppressHydrationWarning>
        <body>
        <TenantProvider>
            <QueryProvider>
                <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
                    <SidebarProvider>
                        <AppSidebar/>
                        <main className="w-full flex h-screen flex-1 flex-col gap-2 pt-2">
                            <header className="flex h-12 shrink-0 items-center gap-2 px-2">
                                <SidebarToggle/>
                                <Separator orientation="vertical" className="mr-2 h-4" />
                                <BreadcrumbBar 
                                    maxItems={5}
                                    maxItemsMobile={2}
                                    showEllipsis={true}
                                    showLoadingStates={true}
                                />
                                <div className="ml-auto">
                                    <ThemeToggle/>
                                </div>
                            </header>
                            <div className="flex flex-1 flex-col overflow-hidden gap-4 p-2 pt-0">
                                <div className="flex flex-1 flex-col overflow-hidden rounded-xl bg-sidebar">
                                    {children}
                                </div>
                            </div>
                        </main>
                    </SidebarProvider>
                </ThemeProvider>
                <Toaster />
            </QueryProvider>
        </TenantProvider>
        </body>
        </html>
    );
}
