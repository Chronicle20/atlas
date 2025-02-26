"use client";

import {createContext, ReactNode, useContext, useEffect, useState} from "react";
import {fetchTenants, Tenant} from "@/lib/tenants";

type TenantContextType = {
    tenants: Tenant[];
    activeTenant: Tenant | null;
    setActiveTenant: (tenant: Tenant) => void;
};

// Create Context
const TenantContext = createContext<TenantContextType | undefined>(undefined);

// Provider Component
export function TenantProvider({children}: { children: ReactNode }) {
    const [activeTenant, setActiveTenant] = useState<Tenant | null>(null);
    const [tenants, setTenants] = useState<Tenant[]>([]);
    const [loading, setLoading] = useState<boolean>(true);
    const [error, setError] = useState(null);

    useEffect(() => {
        console.log("TenantProvider Mounted");

        return () => {
            console.log("TenantProvider Unmounted");
        };
    }, []);


    // Fetch tenants data (you can replace this with your actual data fetching logic)
    useEffect(() => {
        fetchTenants().then((data) => {
            setTenants(data);
            if (!activeTenant) {
                setActiveTenant(data[0] || null); // Set first tenant as default
            }
        })
            .catch((err) => setError(err.message))
            .finally(() => setLoading(false));
    }, []);

    // Set the default active tenants when tenants data is available
    useEffect(() => {
        if (tenants.length > 0) {
            setActiveTenant(tenants[0]); // Set first tenants as default
        }
    }, [tenants]);

    return (
        <TenantContext.Provider value={{tenants, activeTenant, setActiveTenant}}>
            {children}
        </TenantContext.Provider>
    );
}

// Hook to use tenants context
export function useTenant() {
    const context = useContext(TenantContext);
    if (!context) {
        throw new Error("useTenant must be used within a TenantProvider");
    }
    return context;
}
