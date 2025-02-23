import type {Metadata} from "next";
import "./globals.css";
import Sidebar from "@/app/ui/sidebar";

export const metadata: Metadata = {
    title: "AtlasMS",
};

export default function RootLayout({children,}: Readonly<{ children: React.ReactNode; }>) {
    return (
        <html lang="en">
        <body className="bg-nord-0">
        <div className="flex">
            <Sidebar/>
            <main className="flex-1">{children}</main>
        </div>
        </body>
        </html>
    );
}
