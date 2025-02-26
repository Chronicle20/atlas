"use client"

import {useTenant} from "@/context/tenant-context";
import {DataTable} from "@/components/data-table";
import {columns} from "@/app/tenants/columns";

export default function Page() {
    const {tenants} = useTenant()

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
            <div className="items-center justify-between space-y-2">
                <div>
                    <h2 className="text-2xl font-bold tracking-tight">Tenants</h2>
                </div>
            </div>
            <div className="mt-4">
                <DataTable columns={columns} data={tenants}/>
            </div>
        </div>
    );
}