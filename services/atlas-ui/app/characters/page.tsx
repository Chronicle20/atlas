"use client"

import {useTenant} from "@/context/tenant-context";
import {DataTable} from "@/components/data-table";
import {columns} from "@/app/characters/columns";
import {useEffect, useState} from "react";
import {Character, fetchCharacters} from "@/lib/characters";


export default function Page() {
    const {activeTenant} = useTenant();
    const [characters, setCharacters] = useState<Character[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        if (!activeTenant) return;

        setLoading(true);

        fetchCharacters(activeTenant)
            .then((data) => {
                setCharacters(data);
            })
            .catch((err) => setError(err.message))
            .finally(() => setLoading(false));
    }, [activeTenant]);

    if (loading) return <div>Loading...</div>; // Show loading message while fetching data
    if (error) return <div>Error: {error}</div>; // Show error message if fetching failed

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
            <div className="items-center justify-between space-y-2">
                <div>
                    <h2 className="text-2xl font-bold tracking-tight">Characters</h2>
                </div>
            </div>
            <div className="mt-4">
                <DataTable columns={columns} data={characters}/>
            </div>
        </div>
    );
}