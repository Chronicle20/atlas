"use client"

import {useTenant} from "@/context/tenant-context";
import {DataTableWrapper} from "@/components/common/DataTableWrapper";
import {getColumns, hiddenColumns} from "@/app/characters/columns";
import {useCallback, useEffect, useState} from "react";
import {accountsService} from "@/services/api/accounts.service";
import {charactersService} from "@/services/api/characters.service";
import {Character} from "@/types/models/character";
import {Account} from "@/types/models/account";
import {TenantConfig} from "@/types/models/tenant";
import {createErrorFromUnknown} from "@/types/api/errors";


export default function Page() {
    const {activeTenant, fetchTenantConfiguration} = useTenant();
    const [characters, setCharacters] = useState<Character[]>([]);
    const [accounts, setAccounts] = useState<Account[]>([]);
    const [tenantConfig, setTenantConfig] = useState<TenantConfig | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const fetchDataAgain = useCallback(() => {
        if (!activeTenant) return

        setLoading(true)

        Promise.all([
            charactersService.getAll(activeTenant),
            accountsService.getAllAccounts(activeTenant),
            fetchTenantConfiguration(activeTenant.id),
        ])
            .then(([characterData, accountData, tenantConfigData]) => {
                setCharacters(characterData);
                setAccounts(accountData);
                setTenantConfig(tenantConfigData);
            })
            .catch((err: unknown) => {
                const errorInfo = createErrorFromUnknown(err, "Failed to fetch characters data");
                setError(errorInfo.message);
            })
            .finally(() => setLoading(false));
    }, [activeTenant, fetchTenantConfiguration])

    useEffect(() => {
        fetchDataAgain()
    }, [activeTenant, fetchDataAgain])

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
                <DataTableWrapper 
                    columns={columns} 
                    data={characters} 
                    loading={loading}
                    error={error}
                    onRefresh={fetchDataAgain}
                    initialVisibilityState={hiddenColumns}
                    emptyState={{
                        title: "No characters found",
                        description: "There are no characters to display at this time."
                    }}
                />
            </div>
        </div>
    );
}
