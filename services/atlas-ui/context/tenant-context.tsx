"use client";

import {createContext, useContext, useState, ReactNode, useEffect} from "react";

type Tenant = {
    id: string;
    attributes: {
        region: string;
        majorVersion: number;
        minorVersion: number;
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
        // const data = await fetch("/api/configurations/tenants");
        // const json = await data.json();
        setTenants([
            {
                id: "1e800451-586c-40cb-94c1-e277f97e7c2c",
                attributes: {
                    region: "GMS",
                    majorVersion: 12,
                    minorVersion: 1,
                },
            },
            {
                id: "a3fe2199-1bd2-40f7-ba6a-d89d712848b5",
                attributes: {
                    region: "GMS",
                    majorVersion: 83,
                    minorVersion: 1,
                },
            },
            {
                id: "74e95941-84d3-4e12-bf18-67adafb36ba6",
                attributes: {
                    region: "GMS",
                    majorVersion: 87,
                    minorVersion: 1,
                },
            },
            {
                id: "03b7429a-3d62-4f4c-b511-677772853424",
                attributes: {
                    region: "JMS",
                    majorVersion: 185,
                    minorVersion: 1,
                },
            },
        ]); // Assuming the data is under the "data" property in your API response
    };

    // Fetch tenant data (you can replace this with your actual data fetching logic)
    useEffect(() => {
        fetchTenants();
    }, []);

    // Set the default active tenant when tenants data is available
    useEffect(() => {
        if (tenants.length > 0) {
            setActiveTenant(tenants[0]); // Set first tenant as default
        }
    }, [tenants]);

    return (
        <TenantContext.Provider value={{ tenants, activeTenant, setActiveTenant, fetchTenants }}>
            {children}
        </TenantContext.Provider>
    );
}

// Hook to use tenant context
export function useTenant() {
    const context = useContext(TenantContext);
    if (!context) {
        throw new Error("useTenant must be used within a TenantProvider");
    }
    return context;
}
