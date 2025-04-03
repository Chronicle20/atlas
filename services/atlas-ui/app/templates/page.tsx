"use client"

import {DataTable} from "@/components/data-table";
import {columns} from "@/app/templates/columns";
import {useEffect, useState} from "react";
import {fetchTemplates, Template} from "@/lib/templates";

export default function Page() {
    const [templates, setTemplates] = useState<Template[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const fetchDataAgain = () => {
        setLoading(true)
        fetchTemplates()
            .then((data) => setTemplates(data))
            .catch((err) => setError(err.message))
            .finally(() => setLoading(false))
    }

    useEffect(() => {
        fetchDataAgain()
    }, [])

    if (loading) return <div>Loading...</div>; // Show loading message while fetching data
    if (error) return <div>Error: {error}</div>; // Show error message if fetching failed

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
            <div className="items-center justify-between space-y-2">
                <div>
                    <h2 className="text-2xl font-bold tracking-tight">Templates</h2>
                </div>
            </div>
            <div className="mt-4">
                <DataTable columns={columns} data={templates} onRefresh={fetchDataAgain}/>
            </div>
        </div>
    );
}