"use client"

import {useTenant} from "@/context/tenant-context";
import {DataTable} from "@/components/data-table";
import {hiddenColumns} from "@/app/accounts/columns";
import {useEffect, useState} from "react";
import {Account, fetchAccounts} from "@/lib/accounts";
import {getColumns} from "@/app/accounts/columns";
import {Toaster} from "sonner";
import {createErrorFromUnknown} from "@/types/api/errors";


export default function Page() {
    const {activeTenant} = useTenant();
    const [accounts, setAccounts] = useState<Account[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const fetchDataAgain = () => {
        if (!activeTenant) return

        setLoading(true)

        fetchAccounts(activeTenant)
            .then((data) => {
                setAccounts(data)
            })
            .catch((err: unknown) => {
                const errorInfo = createErrorFromUnknown(err, "Failed to fetch accounts");
                setError(errorInfo.message);
            })
            .finally(() => setLoading(false))
    }

    useEffect(() => {
        fetchDataAgain()
    }, [activeTenant])

    if (loading) return <div>Loading...</div>; // Show loading message while fetching data
    if (error) return <div>Error: {error}</div>; // Show error message if fetching failed

    const columns = getColumns({tenant: activeTenant});

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
            <div className="items-center justify-between space-y-2">
                <div>
                    <h2 className="text-2xl font-bold tracking-tight">Accounts</h2>
                </div>
            </div>
            <div className="mt-4">
                <DataTable columns={columns} data={accounts} onRefresh={fetchDataAgain} initialVisibilityState={hiddenColumns}/>
            </div>
            <Toaster richColors/>
        </div>
    );
}