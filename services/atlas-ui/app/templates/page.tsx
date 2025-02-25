"use client"

import {DataTable} from "@/components/data-table";
import {columns} from "@/app/templates/columns";
import {useEffect, useState} from "react";
import {fetchTemplates, Template} from "@/lib/templates";

export default function Page() {
    const [templates, setTemplates] = useState<Template[]>([]);

    useEffect(() => {
        const loadTemplates = async () => {
            try {
                const data = await fetchTemplates();
                setTemplates(data.data);
            } catch (error) {
                console.error(error);
            }
        };
        loadTemplates();
    }, []);

    return (
        <div className="flex flex-col flex-1 container mx-auto p-5 h-full">
            <div className="items-center justify-between space-y-2">
                <div>
                    <h2 className="text-2xl font-bold tracking-tight">Templates</h2>
                </div>
            </div>
            <div className="mt-4">
                <DataTable columns={columns} data={templates}/>
            </div>
        </div>
    );
}