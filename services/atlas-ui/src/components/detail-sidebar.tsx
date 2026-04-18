import { Link } from "react-router-dom";
import { useLocation } from "react-router-dom";

import { cn } from "@/lib/utils"
import {buttonVariants} from "@/components/ui/button";

interface DetailSidebarProps extends React.HTMLAttributes<HTMLElement> {
    items: {
        href: string
        title: string
    }[]
}

export function DetailSidebar({ className, items, ...props }: DetailSidebarProps) {
    const pathname = useLocation().pathname

    return (
        <nav
            className={cn(
                "flex space-x-2 lg:flex-col lg:space-x-0 lg:space-y-1",
                className
            )}
            {...props}
        >
            {items.map((item) => (
                <Link
                    key={item.href}
                    to={item.href}
                    className={cn(
                        buttonVariants({ variant: "ghost" }),
                        pathname === item.href
                            ? "bg-muted hover:bg-muted"
                            : "hover:bg-transparent hover:underline",
                        "justify-start"
                    )}
                >
                    {item.title}
                </Link>
            ))}
        </nav>
    )
}