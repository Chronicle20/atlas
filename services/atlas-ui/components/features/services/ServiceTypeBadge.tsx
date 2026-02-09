"use client";

import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { ServiceType } from "@/types/models/service";

interface ServiceTypeBadgeProps {
  type: ServiceType;
  className?: string;
}

const typeConfig: Record<ServiceType, { label: string; className: string }> = {
  "login-service": {
    label: "Login",
    className: "bg-blue-100 text-blue-800 border-blue-200 dark:bg-blue-900/30 dark:text-blue-300 dark:border-blue-800",
  },
  "channel-service": {
    label: "Channel",
    className: "bg-green-100 text-green-800 border-green-200 dark:bg-green-900/30 dark:text-green-300 dark:border-green-800",
  },
  "drops-service": {
    label: "Drops",
    className: "bg-orange-100 text-orange-800 border-orange-200 dark:bg-orange-900/30 dark:text-orange-300 dark:border-orange-800",
  },
};

/**
 * Badge component for displaying service types with color coding.
 *
 * - Login services: Blue
 * - Channel services: Green
 * - Drops services: Orange
 */
export function ServiceTypeBadge({ type, className }: ServiceTypeBadgeProps) {
  const config = typeConfig[type];

  if (!config) {
    return (
      <Badge variant="outline" className={className}>
        {type}
      </Badge>
    );
  }

  return (
    <Badge
      variant="outline"
      className={cn(config.className, className)}
    >
      {config.label}
    </Badge>
  );
}
