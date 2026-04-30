import { type ReactNode } from "react";
import { useParams } from "react-router-dom";
import { Separator } from "@/components/ui/separator";
import { DetailSidebar } from "@/components/detail-sidebar";

interface TemplateDetailLayoutProps {
    children: ReactNode;
}

export function TemplateDetailLayout({ children }: TemplateDetailLayoutProps) {
    const { id } = useParams();
    const sidebarNavItems = [
        { title: "Global Properties",   href: `/templates/${id}/properties` },
        { title: "Character Templates", href: `/templates/${id}/character/templates` },
        { title: "Character Presets",   href: `/templates/${id}/character/presets` },
        { title: "Socket Handlers",     href: `/templates/${id}/handlers` },
        { title: "Socket Writers",      href: `/templates/${id}/writers` },
        { title: "Worlds",              href: `/templates/${id}/worlds` },
    ];
    return (
        <div className="flex flex-1 flex-col overflow-hidden space-y-6 p-10 pb-16">
            <div className="space-y-0.5">
                <h2 className="text-2xl font-bold tracking-tight">Template Details</h2>
                <p className="text-muted-foreground">{id}</p>
            </div>
            <Separator className="my-6" />
            <div className="flex flex-1 flex-col overflow-hidden space-y-8 lg:flex-row lg:space-x-12 lg:space-y-0">
                <aside className="lg:w-1/5">
                    <DetailSidebar items={sidebarNavItems} />
                </aside>
                <div className="flex-1 overflow-y-auto lg:max-w-4xl">{children}</div>
            </div>
        </div>
    );
}
