"use client";

import {createContext, ReactNode, useContext, useEffect, useState} from "react";
import {fetchTenants, fetchTenantConfiguration} from "@/lib/tenants";
import type {Tenant, TenantConfig} from "@/types/models/tenant";
import {createErrorFromUnknown} from "@/types/api/errors";

type TenantContextType = {
    tenants: Tenant[];
    activeTenant: Tenant | null;
    setActiveTenant: (tenant: Tenant) => void;
    refreshTenants: () => Promise<void>;
    fetchTenantConfiguration: (tenantId: string) => Promise<TenantConfig>;
};

// Create Context
const TenantContext = createContext<TenantContextType | undefined>(undefined);

// Provider Component
export function TenantProvider({children}: { children: ReactNode }) {
    const [activeTenant, setActiveTenantState] = useState<Tenant | null>(null);
    const [tenants, setTenants] = useState<Tenant[]>([]);
    const [loading, setLoading] = useState<boolean>(true);
    const [error, setError] = useState<string | null>(null);

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
            .catch((err: unknown) => {
                const errorInfo = createErrorFromUnknown(err, "Failed to fetch tenants");
                setError(errorInfo.message);
            })
            .finally(() => setLoading(false));
    }, []);

    // Store tenant in localStorage on change
    const setActiveTenant = (tenant: Tenant) => {
        setActiveTenantState(tenant);
        localStorage.setItem(LOCAL_STORAGE_KEY, tenant.id);
    };

    // Function to refresh tenants list
    const refreshTenants = async () => {
        try {
            setLoading(true);
            setError(null); // Clear previous errors
            const data = await fetchTenants();
            setTenants(data);

            // If active tenant was deleted, set a new one
            if (activeTenant && !data.find(t => t.id === activeTenant.id)) {
                const storedId = localStorage.getItem(LOCAL_STORAGE_KEY);
                const storedTenant = data.find((t) => t.id === storedId);
                setActiveTenantState(storedTenant ?? data[0] ?? null);
            }
        } catch (err: unknown) {
            const errorInfo = createErrorFromUnknown(err, "Failed to refresh tenants");
            setError(errorInfo.message);
            console.error("Failed to refresh tenants:", errorInfo);
        } finally {
            setLoading(false);
        }
    };

    // Function to fetch a tenant configuration
    const fetchTenantConfig = async (tenantId: string): Promise<TenantConfig> => {
        try {
            return await fetchTenantConfiguration(tenantId);
        } catch (err: unknown) {
            const errorInfo = createErrorFromUnknown(err, `Failed to fetch configuration for tenant ${tenantId}`);
            throw errorInfo;
        }
    };

    return (
        <TenantContext.Provider value={{
            tenants, 
            activeTenant, 
            setActiveTenant, 
            refreshTenants,
            fetchTenantConfiguration: fetchTenantConfig
        }}>
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
