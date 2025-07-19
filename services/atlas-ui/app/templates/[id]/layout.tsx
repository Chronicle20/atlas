"use client"

import {useParams} from "next/navigation";
import {Separator} from "@/components/ui/separator";
import {DetailSidebar} from "@/components/detail-sidebar";
import {fetchTemplates} from "@/lib/templates";
import type {Template} from "@/types/models/template";
import {useEffect, useState} from "react";
import {ScrollArea} from "@/components/ui/scroll-area";
import {Toaster} from "sonner";
import { LoadingSpinner } from "@/components/common";

interface TemplateDetailLayoutProps {
    children: React.ReactNode
}

export default function TemplateDetailLayout({ children }: TemplateDetailLayoutProps) {
    const { id } = useParams(); // Get templates ID from URL

    const [template, setTemplate] = useState<Template>();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    useEffect(() => {
        if (!id) return; // Ensure id is available

        setLoading(true); // Show loading while fetching

        fetchTemplates()
            .then((data) => {
                const template = data.find((t) => String(t.id) === String(id));
                setTemplate(template);
            })
            .catch((err) => {
                setError(err.message);
            })
            .finally(() => setLoading(false));
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
    ];

    if (loading) return <LoadingSpinner />; // Show loading message while fetching data
    if (error) return <div>Error: {error}</div>; // Show error message if fetching failed

    return (
        <>
            <div className="flex flex-1 flex-col overflow-hidden space-y-6 p-10 pb-16">
                <div className="space-y-0.5">
                    <h2 className="text-2xl font-bold tracking-tight">Template Details</h2>
                    <p className="text-muted-foreground">
                        {template?.id}
                    </p>
                </div>
                <Separator className="my-6" />
                <div className="flex flex-1 flex-col overflow-hidden space-y-8 lg:flex-row lg:space-x-12 lg:space-y-0">
                    <aside className="lg:w-1/7">
                        <DetailSidebar items={sidebarNavItems} />
                    </aside>
                    <ScrollArea className="flex flex-1 flex-col">
                        <div className="flex-1 lg:max-w-4xl">{children}</div>
                    </ScrollArea>
                </div>
            </div>
            <Toaster richColors/>
        </>
    )
}