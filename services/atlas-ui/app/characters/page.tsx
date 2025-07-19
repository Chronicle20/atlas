"use client"

import {useTenant} from "@/context/tenant-context";
import {DataTable} from "@/components/data-table";
import {getColumns, hiddenColumns} from "@/app/characters/columns";
import {useEffect, useState} from "react";
import {Character, fetchCharacters} from "@/lib/characters";
import {Account, fetchAccounts} from "@/lib/accounts";
import {TenantConfig} from "@/lib/tenants";


export default function Page() {
    const {activeTenant, fetchTenantConfiguration} = useTenant();
    const [characters, setCharacters] = useState<Character[]>([]);
    const [accounts, setAccounts] = useState<Account[]>([]);
    const [tenantConfig, setTenantConfig] = useState<TenantConfig | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const fetchDataAgain = () => {
        if (!activeTenant) return

        setLoading(true)

        Promise.all([
            fetchCharacters(activeTenant),
            fetchAccounts(activeTenant),
            fetchTenantConfiguration(activeTenant.id),
        ])
            .then(([characterData, accountData, tenantConfigData]) => {
                setCharacters(characterData);
                setAccounts(accountData);
                setTenantConfig(tenantConfigData);
            })
            .catch((err) => setError(err.message))
            .finally(() => setLoading(false));
    }

    useEffect(() => {
        fetchDataAgain()
    }, [activeTenant])

    if (loading) return <div>Loading...</div>;
    if (error) return <div>Error: {error}</div>;

    const accountMap = new Map(accounts.map(a => [a.id, a]));

    const columns = getColumns({tenant: activeTenant, tenantConfig: tenantConfig, accountMap, onRefresh: fetchDataAgain});

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
            <div className="items-center justify-between space-y-2">
                <div>
                    <h2 className="text-2xl font-bold tracking-tight">Characters</h2>
                </div>
            </div>
            <div className="mt-4">
                <DataTable columns={columns} data={characters} onRefresh={fetchDataAgain}
                           initialVisibilityState={hiddenColumns}/>
            </div>
        </div>
    );
}
