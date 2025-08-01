"use client"

import {useTenant} from "@/context/tenant-context";
import {DataTableWrapper} from "@/components/common/DataTableWrapper";
import {hiddenColumns} from "@/app/accounts/columns";
import {useCallback, useEffect, useState} from "react";
import {accountsService} from "@/services/api/accounts.service";
import {Account} from "@/types/models/account";
import {getColumns} from "@/app/accounts/columns";
import {Toaster} from "sonner";
import {createErrorFromUnknown} from "@/types/api/errors";


export default function Page() {
    const {activeTenant} = useTenant();
    const [accounts, setAccounts] = useState<Account[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const fetchDataAgain = useCallback(() => {
        if (!activeTenant) return

        setLoading(true)

        accountsService.getAllAccounts(activeTenant)
            .then((data) => {
                setAccounts(data)
            })
            .catch((err: unknown) => {
                const errorInfo = createErrorFromUnknown(err, "Failed to fetch accounts");
                setError(errorInfo.message);
            })
            .finally(() => setLoading(false))
    }, [activeTenant])

    useEffect(() => {
        fetchDataAgain()
    }, [activeTenant, fetchDataAgain])

    const columns = getColumns({tenant: activeTenant});

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
            <div className="items-center justify-between space-y-2">
                <div>
                    <h2 className="text-2xl font-bold tracking-tight">Accounts</h2>
                </div>
            </div>
            <div className="mt-4">
                <DataTableWrapper 
                    columns={columns} 
                    data={accounts} 
                    loading={loading}
                    error={error}
                    onRefresh={fetchDataAgain} 
                    initialVisibilityState={hiddenColumns}
                    emptyState={{
                        title: "No accounts found",
                        description: "There are no accounts to display at this time."
                    }}
                />
            </div>
            <Toaster richColors/>
        </div>
    );
}