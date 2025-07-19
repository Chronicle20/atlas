"use client"

import {useTenant} from "@/context/tenant-context";
import {DataTable} from "@/components/data-table";
import {hiddenColumns} from "@/app/guilds/columns";
import {useEffect, useState} from "react";
import {fetchGuilds} from "@/lib/guilds";
import {getColumns} from "@/app/guilds/columns";
import {Toaster} from "sonner";
import {fetchCharacters} from "@/lib/characters";
import {Guild} from "@/types/models/guild";
import {Character} from "@/types/models/character";
import {TenantConfig} from "@/types/models/tenant";
import {createErrorFromUnknown} from "@/types/api/errors";
import {PageLoader} from "@/components/common/PageLoader";
import {ErrorDisplay} from "@/components/common/ErrorDisplay";


export default function Page() {
    const {activeTenant, fetchTenantConfiguration} = useTenant();
    const [guilds, setGuilds] = useState<Guild[]>([]);
    const [characters, setCharacters] = useState<Character[]>([]);
    const [tenantConfig, setTenantConfig] = useState<TenantConfig | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const fetchDataAgain = () => {
        if (!activeTenant) return

        setLoading(true)

        Promise.all([
            fetchGuilds(activeTenant),
            fetchCharacters(activeTenant),
            fetchTenantConfiguration(activeTenant.id),
        ])
            .then(([guildData, characterData, tenantConfigData]) => {
                setGuilds(guildData);
                setCharacters(characterData);
                setTenantConfig(tenantConfigData);
            })
            .catch((err: unknown) => {
                const errorInfo = createErrorFromUnknown(err, "Failed to fetch guilds and tenant config");
                setError(errorInfo.message);
            })
            .finally(() => setLoading(false));
    }

    useEffect(() => {
        fetchDataAgain()
    }, [activeTenant])

    if (loading) return <PageLoader />;
    if (error) return <ErrorDisplay error={error} retry={fetchDataAgain} />;

    const characterMap = new Map(characters.map(c => [c.id, c]));

    const columns = getColumns({ tenant: tenantConfig, characterMap });

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
