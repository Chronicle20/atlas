"use client"

import {useParams} from "next/navigation";
import {Separator} from "@/components/ui/separator";
import {DetailSidebar} from "@/components/detail-sidebar";
import {fetchTemplates, Template} from "@/lib/templates";
import {useEffect, useState} from "react";

interface TemplateDetailLayoutProps {
    children: React.ReactNode
}

export default function TemplateDetailLayout({ children }: TemplateDetailLayoutProps) {
    const { id } = useParams(); // Get templates ID from URL

    const [template, setTemplate] = useState<Template>();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    useEffect(() => {
        const loadTemplates = async () => {
            if (!id) return; // Ensure id is available

            setLoading(true); // Show loading while fetching

            try {
                const data: Template[] = await fetchTemplates();

                const template = data.data.find((t) => String(t.id) === String(id));
                setTemplate(template);
            } catch (err) {
                setError(err.message);
            } finally {
                setLoading(false);
            }
        };

        loadTemplates();
    }, [id]);

    const sidebarNavItems = [
        {
            title: "Global Properties",
            href: "/templates/" + id + "/properties",
        },
        {
            title: "Character Templates",
            href: "/templates/" + id + "/character/templates",
        },
        {
            title: "Socket Handlers",
            href: "/templates/" + id + "/handlers",
        },
        {
            title: "Socket Writers",
            href: "/templates/" + id + "/writers",
        },
        {
            title: "Worlds",
            href: "/templates/" + id + "/worlds",
        },
    ]

    return (
        <>
            <div className="hidden space-y-6 p-10 pb-16 md:block">
                <div className="space-y-0.5">
                    <h2 className="text-2xl font-bold tracking-tight">Template Details</h2>
                    <p className="text-muted-foreground">
                        {template?.id}
                    </p>
                </div>
                <Separator className="my-6" />
                <div className="flex flex-col space-y-8 lg:flex-row lg:space-x-12 lg:space-y-0">
                    <aside className="-mx-4 lg:w-1/5">
                        <DetailSidebar items={sidebarNavItems} />
                    </aside>
                    <div className="flex-1 lg:max-w-4xl">{children}</div>
                </div>
            </div>
        </>
    )
}