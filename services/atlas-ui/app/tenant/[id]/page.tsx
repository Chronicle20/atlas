"use client"

import {useTenant} from "@/context/tenant-context";
import {useParams} from "next/navigation";

export default function Page() {
    const { id } = useParams(); // Get tenant ID from URL
    const {tenants} = useTenant()
    const tenant = tenants.find((t) => t.id === id);

    return (
        <div className="flex flex-col flex-1 container mx-auto p-5 h-full">
            <div className="items-center justify-between space-y-2">
                <div>
                    <h2 className="text-2xl font-bold tracking-tight">Tenant Details</h2>
                </div>
            </div>
            <div className="mt-4">
                <p><strong>ID:</strong> {tenant?.id}</p>
                <p><strong>Region:</strong> {tenant?.attributes.region}</p>
                <p><strong>Version:</strong> {tenant?.attributes.majorVersion}.{tenant?.attributes.minorVersion}</p>
            </div>
        </div>
    );
}