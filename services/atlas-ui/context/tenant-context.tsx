"use client";

import {createContext, useContext, useState, ReactNode, useEffect} from "react";

type Tenant = {
    id: string;
    attributes: {
        region: string;
        majorVersion: number;
        minorVersion: number;
        usesPin: boolean;
        worlds: {
            name: string;
            flag: string;
            serverMessage: string;
            eventMessage: string;
            whyAmIRecommended: string;
        }[];
    };
};

type TenantContextType = {
    tenants: Tenant[];
    activeTenant: Tenant | null;
    setActiveTenant: (tenant: Tenant) => void;
    fetchTenants: () => Promise<void>;
};

// Create Context
const TenantContext = createContext<TenantContextType | undefined>(undefined);

// Provider Component
export function TenantProvider({ children }: { children: ReactNode }) {
    const [activeTenant, setActiveTenant] = useState<Tenant | null>(null);
    const [tenants, setTenants] = useState<Tenant[]>([]);

    // Simulate an API call to fetch tenants
    const fetchTenants = async () => {
        const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || "http://localhost:3000";
        const response = await fetch(rootUrl + "/api/configurations/tenants");
        const data = await response.json();
        if (data?.data) {
            setTenants(data.data);
        }
    };

    // Fetch tenants data (you can replace this with your actual data fetching logic)
    useEffect(() => {
        fetchTenants();
    }, []);

    // Set the default active tenants when tenants data is available
    useEffect(() => {
        if (tenants.length > 0) {
            setActiveTenant(tenants[0]); // Set first tenants as default
        }
    }, [tenants]);

    return (
        <TenantContext.Provider value={{ tenants, activeTenant, setActiveTenant, fetchTenants }}>
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
