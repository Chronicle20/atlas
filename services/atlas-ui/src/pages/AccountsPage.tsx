import { useTenant } from "@/context/tenant-context";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { hiddenColumns, getColumns } from "@/pages/accounts-columns";
import { useCallback, useEffect, useState } from "react";
import { useAccounts, useInvalidateAccounts } from "@/lib/hooks/api/useAccounts";
import { bansService } from "@/services/api/bans.service";
import type { Account } from "@/types/models/account";
import { BanType, type Ban, type CheckBanAttributes } from "@/types/models/ban";
import { CreateBanDialog } from "@/components/features/bans/CreateBanDialog";
import { DeleteBanDialog } from "@/components/features/bans/DeleteBanDialog";
import { Toaster, toast } from "sonner";
import { AccountPageSkeleton } from "@/components/common/skeletons/AccountPageSkeleton";

export function AccountsPage() {
  const { activeTenant } = useTenant();
  const accountsQuery = useAccounts(activeTenant!);
  const { invalidateAll: invalidateAccounts } = useInvalidateAccounts();

  const accounts = accountsQuery.data ?? [];
  const loading = accountsQuery.isLoading;
  const error = accountsQuery.error?.message ?? null;

  const [banStatuses, setBanStatuses] = useState<Map<string, CheckBanAttributes>>(new Map());
  const [banStatusLoading, setBanStatusLoading] = useState(false);
  const [createBanDialogOpen, setCreateBanDialogOpen] = useState(false);
  const [deleteBanDialogOpen, setDeleteBanDialogOpen] = useState(false);
  const [selectedAccount, setSelectedAccount] = useState<Account | null>(null);
  const [banToDelete, setBanToDelete] = useState<Ban | null>(null);

  // Ban statuses are fetched once per account list with bounded concurrency.
  // Kept as a side effect rather than a query because the fan-out and
  // fail-open semantics don't map cleanly onto a single React Query entry.
  const fetchBanStatuses = useCallback(
    async (accountList: Account[]) => {
      if (!activeTenant || accountList.length === 0) return;

      setBanStatusLoading(true);
      const statuses = new Map<string, CheckBanAttributes>();
      const concurrency = 3;

      for (let i = 0; i < accountList.length; i += concurrency) {
        const batch = accountList.slice(i, i + concurrency);
        const results = await Promise.allSettled(
          batch.map(async (account) => {
            const result = await bansService.checkBan(activeTenant, { accountId: Number(account.id) });
            return { accountId: account.id, result };
          })
        );

        for (const result of results) {
          if (result.status === "fulfilled") {
            statuses.set(result.value.accountId, result.value.result.attributes);
          }
        }
      }

      setBanStatuses(statuses);
      setBanStatusLoading(false);
    },
    [activeTenant]
  );

  useEffect(() => {
    if (accounts.length > 0) {
      fetchBanStatuses(accounts);
    }
  }, [accounts, fetchBanStatuses]);

  const handleBanAccount = (account: Account) => {
    setSelectedAccount(account);
    setCreateBanDialogOpen(true);
  };

  const handleRemoveBan = async (account: Account) => {
    if (!activeTenant) return;

    try {
      const bans = await bansService.getBansByType(activeTenant, BanType.Account);
      const matchingBan = bans.find(b => b.attributes.value === account.id);

      if (matchingBan) {
        setBanToDelete(matchingBan);
        setDeleteBanDialogOpen(true);
      } else {
        toast.error("Could not find an active ban for this account");
      }
    } catch (err: unknown) {
      toast.error("Failed to look up ban: " + (err instanceof Error ? err.message : "Unknown error"));
    }
  };

  const handleBanCreated = () => {
    setSelectedAccount(null);
    fetchBanStatuses(accounts);
  };

  const handleBanDeleted = () => {
    setBanToDelete(null);
    fetchBanStatuses(accounts);
  };

  const columns = getColumns({
    tenant: activeTenant,
    onRefresh: () => invalidateAccounts(),
    banStatuses,
    banStatusLoading,
    onBanAccount: handleBanAccount,
    onRemoveBan: handleRemoveBan,
  });

  if (loading) {
    return <AccountPageSkeleton />;
  }

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="items-center justify-between space-y-2">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Accounts</h2>
        </div>
      </div>
      <div className="mt-4">
        <DataTableWrapper
          columns={columns}
          data={accounts}
          error={error}
          onRefresh={() => invalidateAccounts()}
          initialVisibilityState={hiddenColumns}
          emptyState={{
            title: "No accounts found",
            description: "There are no accounts to display at this time.",
          }}
        />
      </div>

      <CreateBanDialog
        open={createBanDialogOpen}
        onOpenChange={setCreateBanDialogOpen}
        tenant={activeTenant}
        onSuccess={handleBanCreated}
        prefill={selectedAccount ? {
          banType: BanType.Account,
          value: selectedAccount.id,
        } : undefined}
      />

      <DeleteBanDialog
        ban={banToDelete}
        open={deleteBanDialogOpen}
        onOpenChange={setDeleteBanDialogOpen}
        tenant={activeTenant}
        onSuccess={handleBanDeleted}
      />

      <Toaster richColors />
    </div>
  );
}
