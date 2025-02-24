"use client"

import {useSidebar} from "@/components/ui/sidebar"
import {Button} from "@/components/ui/button";
import {PanelLeft} from "lucide-react";
import * as React from "react";

export function SidebarToggle() {
    const {toggleSidebar} = useSidebar()

    return (
        <Button variant="outline" size="icon" onClick={toggleSidebar}>
            <PanelLeft className="h-[1.2rem] w-[1.2rem] rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0"/>
            <PanelLeft className="absolute h-[1.2rem] w-[1.2rem] rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100"/>
            <span className="sr-only">Toggle theme</span>
        </Button>
    )
}
