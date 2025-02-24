"use client";

import {createContext, useContext, useState, ReactNode, useEffect, useRef} from "react";

type Tenant = {
    id: string;
    attributes: {
        region: string;
        majorVersion: number;
        minorVersion: number;
        usesPin: boolean;
        characters: {
            templates: {
                jobIndex: number;
                subJobIndex: number;
                gender: number;
                mapId: number;
                faces: number[];
                hairs: number[];
                hairColors: number[];
                skinColors: number[];
                tops: number[];
                bottoms: number[];
                shoes: number[];
                weapons: number[];
                items: number[];
                skills: number[];
            }[];
        };
        npcs: {
            npcId: number;
            impl: string;
        }[];
        socket: {
            handlers: {
                opCode: string;
                validator: string;
                handler: string;
                options: any;
            }[];
            writers: {
                opCode: string;
                writer: string;
                options: any;
            }[];
        }
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
    updateTenant: (tenant: Tenant | undefined, updatedAttributes: any) => Promise<Tenant>;
};

// Create Context
const TenantContext = createContext<TenantContextType | undefined>(undefined);

// Provider Component
export function TenantProvider({children}: { children: ReactNode }) {
    const [activeTenant, setActiveTenant] = useState<Tenant | null>(null);
    const [tenants, setTenants] = useState<Tenant[]>([]);
    const [loading, setLoading] = useState<boolean>(true);
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [error, setError] = useState(null);

    const hasFetched = useRef(false);

    // Simulate an API call to fetch tenants
    const fetchTenants = async () => {
        if (hasFetched.current) return;
        hasFetched.current = true;

        setLoading(true);
        try {
            const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || "http://localhost:3000";
            const response = await fetch(rootUrl + "/api/configurations/tenants");
            const data = await response.json();
            if (data?.data) {
                setTenants(data.data);
                if (!activeTenant) {
                    setActiveTenant(data.data[0] || null); // Set first tenant as default
                }
            }
        } catch (error) {
            console.error("Error fetching tenants:", error);
        } finally {
            setLoading(false);
        }
    };

    const updateTenant = async (tenant : Tenant | undefined, updatedAttributes) => {
        if (!tenant) return;

        setIsSubmitting(true);
        try {
            const input = {
                data: {
                    id: tenant.id,
                    type: "tenants",
                    attributes: {
                        ...tenant.attributes,
                        ...updatedAttributes,
                    },
                },
            };

            const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || "http://localhost:3000";
            const response = await fetch(`${rootUrl}/api/configurations/tenants/${tenant.id}`, {
                method: "PATCH",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(input),
            });

            if (!response.ok) throw new Error("Failed to submit data.");

            // If the request is successful, update the local tenant state
            return { ...tenant, attributes: { ...tenant.attributes, ...updatedAttributes } };
            // setTenant(updatedTenant);
        } catch (err) {
            setError(err.message);
        } finally {
            setIsSubmitting(false);
        }
    };

    useEffect(() => {
        console.log("TenantProvider Mounted");

        return () => {
            console.log("TenantProvider Unmounted");
        };
    }, []);


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
        <TenantContext.Provider value={{tenants, activeTenant, setActiveTenant, fetchTenants, updateTenant}}>
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
