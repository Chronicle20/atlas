"use client"

import {useTenant} from "@/context/tenant-context";

export default function Home() {
    const {activeTenant} = useTenant()

    return (
        <div>
            {activeTenant?.id}
        </div>
    );
}
