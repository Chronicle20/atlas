import { useEffect, useState } from "react";
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { accountsService } from "@/services/api/accounts.service";
import type { Account } from "@/types/models/account";
import type { Tenant } from "@/types/models/tenant";

export interface UseAccountByNameOptions {
  pollUntilFound?: boolean;
  timeoutMs?: number;
  intervalMs?: number;
}

export interface UseAccountByNameResult {
  query: UseQueryResult<Account[], Error>;
  timedOut: boolean;
}

export const accountByNameKeys = {
  all: ["account", "by-name"] as const,
  query: (tenantId: string | undefined, name: string) =>
    [...accountByNameKeys.all, tenantId, name] as const,
};

export function useAccountByName(
  tenant: Tenant,
  name: string,
  options: UseAccountByNameOptions = {},
): UseAccountByNameResult {
  const interval = options.intervalMs ?? 1000;
  const timeout = options.timeoutMs ?? 30000;
  const [timedOut, setTimedOut] = useState(false);

  const query = useQuery({
    queryKey: accountByNameKeys.query(tenant?.id, name),
    queryFn: () => accountsService.getAllAccounts({ name }),
    enabled: !!tenant?.id && !!name && !timedOut,
    refetchInterval: ({ state }) => {
      if (timedOut || !options.pollUntilFound) return false;
      const found = Array.isArray(state.data) && (state.data as Account[]).length > 0;
      return found ? false : interval;
    },
  });

  useEffect(() => {
    if (!options.pollUntilFound) return;
    setTimedOut(false);
    const t = setTimeout(() => setTimedOut(true), timeout);
    return () => clearTimeout(t);
  }, [options.pollUntilFound, timeout, name]);

  return { query, timedOut };
}
