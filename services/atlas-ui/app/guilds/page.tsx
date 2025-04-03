"use client"

import {useTenant} from "@/context/tenant-context";
import {DataTable} from "@/components/data-table";
import {hiddenColumns} from "@/app/guilds/columns";
import {useEffect, useState} from "react";
import {Guild, fetchGuilds} from "@/lib/guilds";
import {getColumns} from "@/app/guilds/columns";
import {Toaster} from "sonner";
import {Character, fetchCharacters} from "@/lib/characters";


export default function Page() {
    const {activeTenant} = useTenant();
    const [guilds, setGuilds] = useState<Guild[]>([]);
    const [characters, setCharacters] = useState<Character[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const fetchDataAgain = () => {
        if (!activeTenant) return

        setLoading(true)

        Promise.all([
            fetchGuilds(activeTenant),
            fetchCharacters(activeTenant),
        ])
            .then(([guildData, characterData]) => {
                setGuilds(guildData);
                setCharacters(characterData);
            })
            .catch((err) => setError(err.message))
            .finally(() => setLoading(false));
    }

    useEffect(() => {
        fetchDataAgain()
    }, [activeTenant])

    if (loading) return <div>Loading...</div>;
    if (error) return <div>Error: {error}</div>;

    const characterMap = new Map(characters.map(c => [c.id, c]));

    const columns = getColumns({ tenant: activeTenant, characterMap });

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
            <div className="items-center justify-between space-y-2">
                <div>
                    <h2 className="text-2xl font-bold tracking-tight">Guilds</h2>
                </div>
            </div>
            <div className="mt-4">
                <DataTable columns={columns} data={guilds} onRefresh={fetchDataAgain} initialVisibilityState={hiddenColumns}/>
            </div>
            <Toaster richColors/>
        </div>
    );
}