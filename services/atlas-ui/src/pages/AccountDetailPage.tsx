import { useState } from "react";
import { useParams } from "react-router-dom";
import { Toaster } from "@/components/ui/sonner";
import { useTenant } from "@/context/tenant-context";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useAccount } from "@/lib/hooks/api/useAccounts";
import { useWallet } from "@/lib/hooks/api/useWallet";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { ErrorDisplay } from "@/components/common";
import { AccountDetailSkeleton } from "@/components/common/skeletons/AccountDetailSkeleton";
import { WalletPanel } from "@/components/features/accounts/WalletPanel";
import { ApplyPresetDialog } from "@/components/features/characters/ApplyPresetDialog";

function getLoginStateName(state: number): string {
  if (state === 0) return "Logged Out";
  if (state === 1) return "Logged In";
  return "In Transition";
}

function getLoginStateBadgeVariant(state: number): "secondary" | "default" | "outline" {
  if (state === 0) return "secondary";
  if (state === 1) return "default";
  return "outline";
}

export function AccountDetailPage() {
  const { id } = useParams();
  const { activeTenant } = useTenant();
  const [applyOpen, setApplyOpen] = useState(false);

  const accountQuery = useAccount(activeTenant!, id ?? "");
  const walletQuery = useWallet(activeTenant!, id ?? "");
  const tenantConfigQuery = useTenantConfiguration(activeTenant?.id ?? "");
  const hasPresets =
    (tenantConfigQuery.data?.attributes?.characters?.presets ?? []).length > 0;

  const account = accountQuery.data ?? null;
  const wallet = walletQuery.data ?? null;
  const loading = accountQuery.isLoading || walletQuery.isLoading;
  const error = accountQuery.error?.message ?? null;
  const walletError = walletQuery.error ? "Wallet not found for this account" : null;

  if (loading) return <AccountDetailSkeleton />;
  if (error || !account) return <ErrorDisplay error={error || "Account not found"} className="p-4" />;

  const lastLogin = account.attributes.lastLogin
    ? new Date(account.attributes.lastLogin).toLocaleString()
    : "Never";

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16 h-screen overflow-auto">
      <div className="flex items-center justify-between space-y-2">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">{account.attributes.name}</h2>
        </div>
        <div>
          {activeTenant && hasPresets && (
            <Button onClick={() => setApplyOpen(true)}>Add character from preset</Button>
          )}
        </div>
      </div>
      {activeTenant && (
        <ApplyPresetDialog
          tenant={activeTenant}
          accountId={Number(account.id)}
          open={applyOpen}
          onOpenChange={setApplyOpen}
        />
      )}

      <div className="flex flex-row gap-6">
        <Card className="flex-1">
          <CardHeader>
            <CardTitle>Account Information</CardTitle>
          </CardHeader>
          <CardContent className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <p className="text-muted-foreground">Account ID</p>
              <p className="font-medium">{account.id}</p>
            </div>
            <div>
              <p className="text-muted-foreground">Name</p>
              <p className="font-medium">{account.attributes.name}</p>
            </div>
            <div>
              <p className="text-muted-foreground">Gender</p>
              <p className="font-medium">{account.attributes.gender === 0 ? "Male" : "Female"}</p>
            </div>
            <div>
              <p className="text-muted-foreground">State</p>
              <Badge variant={getLoginStateBadgeVariant(account.attributes.loggedIn)}>
                {getLoginStateName(account.attributes.loggedIn)}
              </Badge>
            </div>
            <div>
              <p className="text-muted-foreground">Character Slots</p>
              <p className="font-medium">{account.attributes.characterSlots}</p>
            </div>
            <div>
              <p className="text-muted-foreground">Terms of Service</p>
              <p className="font-medium">{account.attributes.tos ? "Accepted" : "Not Accepted"}</p>
            </div>
            <div>
              <p className="text-muted-foreground">Last Login</p>
              <p className="font-medium">{lastLogin}</p>
            </div>
            <div>
              <p className="text-muted-foreground">PIN Attempts</p>
              <p className="font-medium">{account.attributes.pinAttempts}</p>
            </div>
            <div>
              <p className="text-muted-foreground">PIC Attempts</p>
              <p className="font-medium">{account.attributes.picAttempts}</p>
            </div>
          </CardContent>
        </Card>

        {wallet && activeTenant ? (
          <WalletPanel wallet={wallet} accountId={String(id)} tenant={activeTenant} />
        ) : (
          <Card className="flex-1">
            <CardHeader>
              <CardTitle>Cash Shop Wallet</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                {walletError || "No wallet associated with this account."}
              </p>
            </CardContent>
          </Card>
        )}
      </div>

      <Toaster richColors />
    </div>
  );
}
