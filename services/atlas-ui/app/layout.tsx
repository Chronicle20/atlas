import type {Metadata} from "next";
import "./globals.css";
import {SidebarProvider} from "@/components/ui/sidebar"
import {AppSidebar} from "@/components/app-sidebar"
import {ThemeProvider} from "next-themes";
import {ThemeToggle} from "@/components/theme-toggle";
import {SidebarToggle} from "@/components/sidebar-toggle";
import {TenantProvider} from "@/context/tenant-context";
import {Toaster} from "@/components/ui/sonner";

export const metadata: Metadata = {
    title: "AtlasMS",
};

export default function RootLayout({children,}: Readonly<{ children: React.ReactNode; }>) {
    return (
        <html lang="en" suppressHydrationWarning>
        <body>
        <TenantProvider>
            <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
                <SidebarProvider>
                    <AppSidebar/>
                    <main className="w-full flex h-screen flex-1 flex-col gap-2 pt-2">
                        <div className="flex items-center h-12 justify-between px-2">
                            <SidebarToggle/>
                            <ThemeToggle/>
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
        </TenantProvider>
        </body>
        </html>
    );
}
