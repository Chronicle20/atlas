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
    const [activeTenant, setActiveTenantState] = useState<Tenant | null>(null);
    const [tenants, setTenants] = useState<Tenant[]>([]);
    const [loading, setLoading] = useState<boolean>(true);
    const [error, setError] = useState(null);

    const LOCAL_STORAGE_KEY = "activeTenantId";

    // Fetch tenants data (you can replace this with your actual data fetching logic)
    useEffect(() => {
        fetchTenants()
            .then((data) => {
                setTenants(data);

                const storedId = localStorage.getItem(LOCAL_STORAGE_KEY);
                const storedTenant = data.find((t) => t.id === storedId);

                // Prefer localStorage value, fallback to first tenant
                setActiveTenantState(storedTenant ?? data[0] ?? null);
            })
            .catch((err) => setError(err.message))
            .finally(() => setLoading(false));
    }, []);

    // Store tenant in localStorage on change
    const setActiveTenant = (tenant: Tenant) => {
        setActiveTenantState(tenant);
        localStorage.setItem(LOCAL_STORAGE_KEY, tenant.id);
    };

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
