"use client"

import {useTenant} from "@/context/tenant-context";
import {DataTable} from "@/components/data-table";
import {columns} from "@/app/templates/columns";
import {useEffect, useState} from "react";

interface Template {
    id: string;
    attributes: {
        region: string;
        majorVersion: number;
        minorVersion: number;
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
}

export default function Page() {
    const [templates, setTemplates] = useState<Template[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const fetchTemplates = async () => {
            try {
                const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || "http://localhost:3000";
                const response = await fetch(rootUrl + "/api/configurations/templates", {
                    method: "GET",
                    headers: {
                        "Content-Type": "application/json",
                    },
                });
                if (!response.ok) {
                    throw new Error("Failed to fetch templates.");
                }
                const data = await response.json();
                setTemplates(data.data);
            } catch (err: any) {
                setError(err.message);
            } finally {
                setLoading(false);
            }
        };

        fetchTemplates();
    }, []);

    return (
        <div className="flex flex-col flex-1 container mx-auto p-5 h-full">
            <div className="items-center justify-between space-y-2">
                <div>
                    <h2 className="text-2xl font-bold tracking-tight">Templates</h2>
                </div>
            </div>
            <div className="mt-4">
                <DataTable columns={columns} data={templates}/>
            </div>
        </div>
    );
}