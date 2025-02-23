"use client"

import {useState} from "react";
import Link from "next/link";
import {ChevronDown, ChevronRight} from "lucide-react";

const menuItems = [
    {
        title: "Dashboard",
        children: [
            {title: "Overview", href: "/dashboard/overview"},
        ],
    },
    {
        title: "Operations",
        children: [
            {title: "Accounts", href: "/account"},
            {title: "Characters", href: "/character"},
            {title: "Drops", href: "/drop"},
        ],
    },
    {
        title: "Administration",
        children: [
            {title: "Tenants", href: "/tenant"},
        ],
    },
];

export default function Sidebar() {
    const [openMenus, setOpenMenus] = useState({});

    const toggleMenu = (title) => {
        setOpenMenus((prev) => ({...prev, [title]: !prev[title]}));
    };

    return (
        <div className="w-[201px] h-screen bg-nord-0 text-nord-9 flex flex-col border-r border-nord-2 pr-px">
            <div className="w-[200px] h-[200px] bg-nord-0 flex items-center justify-center">
                <Link key="/" href="/">
                    LOGO
                </Link>
            </div>
            <nav>
                {menuItems.map((item) => (
                    <div className="bg-nord-1" key={item.title}>
                        <button
                            className="flex items-center justify-between w-full p-3 text-left hover:bg-nord-2 hover:text-nord-5"
                            onClick={() => toggleMenu(item.title)}
                        >
                            {item.title}
                            {openMenus[item.title] ? <ChevronDown size={16}/> : <ChevronRight size={16}/>}
                        </button>
                        {openMenus[item.title] && (
                            <div className="">
                                {item.children.map((subItem) => (
                                    <Link key={subItem.href} href={subItem.href}>
                                        <div className="pl-6 pr-3 pt-3 pb-3 text-sm hover:bg-nord-2 hover:text-nord-5">{subItem.title}</div>
                                    </Link>
                                ))}
                            </div>
                        )}
                    </div>
                ))}
            </nav>
        </div>
    );
}
