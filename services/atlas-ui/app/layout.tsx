import type {Metadata} from "next";
import "./globals.css";
import {SidebarProvider} from "@/components/ui/sidebar"
import {AppSidebar} from "@/components/app-sidebar"
import {ThemeProvider} from "next-themes";
import {ThemeToggle} from "@/components/theme-toggle";
import {SidebarToggle} from "@/components/sidebar-toggle";

export const metadata: Metadata = {
    title: "AtlasMS",
};

export default function RootLayout({children,}: Readonly<{ children: React.ReactNode; }>) {
    return (
        <html lang="en">
        <body>
        <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
            <SidebarProvider>
                <AppSidebar/>
                <main className="w-full">
                    <div className="flex items-center h-12 justify-between px-2">
                        <SidebarToggle/>
                        <ThemeToggle/>
                    </div>
                    {children}
                </main>
            </SidebarProvider>
        </ThemeProvider>
        </body>
        </html>
    );
}
