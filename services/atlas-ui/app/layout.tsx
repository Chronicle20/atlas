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
                            <div className="flex items-center h-12 justify-between px-2">
                                <SidebarToggle/>
                                <ThemeToggle/>
                            </div>
                            <div className="px-2">
                                <BreadcrumbBar 
                                    className="mb-2" 
                                    maxItems={5}
                                    maxItemsMobile={2}
                                    showEllipsis={true}
                                    showLoadingStates={true}
                                />
                            </div>
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
