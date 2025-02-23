"use client";

import { useEffect, useState } from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

interface Tenant {
    id: string;
    attributes: {
        region: string;
    };
}

export default function AppSidebarFooter() {
    const [tenants, setTenants] = useState<Tenant[]>([]);
    const [selectedRegion, setSelectedRegion] = useState<string | null>(null);

    useEffect(() => {
        async function fetchTenants() {
            try {
                const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || "http://localhost:3000";
                const response = await fetch(rootUrl + "/api/configurations/tenants", {
                    method: "GET",
                    headers: {
                        "Content-Type": "application/vnd.api+json",
                    },
                    mode: "no-cors",
                });
                const data = await response.json();
                if (data?.data) {
                    setTenants(data.data);
                }
            } catch (error) {
                console.error("Failed to fetch tenants:", error);
            }
        }

        fetchTenants();
    }, []);

    return (
        <div className="p-4 border-t">
            <Select onValueChange={(value) => setSelectedRegion(value)}>
                <SelectTrigger className="w-full">
                    <SelectValue placeholder="Select a Tenant" />
                </SelectTrigger>
                <SelectContent>
                    {tenants.map((tenant) => (
                        <SelectItem key={tenant.id} value={tenant.attributes.region}>
                            {tenant.id}
                        </SelectItem>
                    ))}
                </SelectContent>
            </Select>
        </div>
    );
}
