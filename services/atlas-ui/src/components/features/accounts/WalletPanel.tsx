import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Plus, Wallet } from "lucide-react";
import { toast } from "sonner";
import type { Wallet as WalletModel } from "@/types/models/wallet";
import type { Tenant } from "@/types/models/tenant";
import { useUpdateWallet } from "@/lib/hooks/api/useWallet";

interface WalletPanelProps {
    wallet: WalletModel;
    accountId: string;
    tenant: Tenant;
}

type CurrencyType = "credit" | "points" | "prepaid";

const currencyLabels: Record<CurrencyType, string> = {
    credit: "NX Credit",
    points: "Maple Points",
    prepaid: "Prepaid NX",
};

export function WalletPanel({ wallet, accountId, tenant }: WalletPanelProps) {
    const [dialogOpen, setDialogOpen] = useState(false);
    const [activeCurrency, setActiveCurrency] = useState<CurrencyType>("credit");
    const [amount, setAmount] = useState("");
    const updateWallet = useUpdateWallet();

    const openAddDialog = (currency: CurrencyType) => {
        setActiveCurrency(currency);
        setAmount("");
        setDialogOpen(true);
    };

    const handleAddBalance = () => {
        const parsed = parseInt(amount, 10);
        if (isNaN(parsed) || parsed <= 0) {
            toast.error("Please enter a valid positive amount");
            return;
        }

        const newBalances = {
            credit: wallet.attributes.credit,
            points: wallet.attributes.points,
            prepaid: wallet.attributes.prepaid,
        };
        newBalances[activeCurrency] += parsed;

        updateWallet.mutate(
            {
                tenant,
                accountId,
                credit: newBalances.credit,
                points: newBalances.points,
                prepaid: newBalances.prepaid,
            },
            {
                onSuccess: () => {
                    toast.success(`Added ${parsed} ${currencyLabels[activeCurrency]}`);
                    setDialogOpen(false);
                },
                onError: (error) => {
                    toast.error("Failed to update wallet: " + (error instanceof Error ? error.message : "Unknown error"));
                },
            }
        );
    };

    return (
        <>
            <Card className="flex-1">
                <CardHeader>
                    <div className="flex items-center gap-2">
                        <Wallet className="h-5 w-5" />
                        <CardTitle>Cash Shop Wallet</CardTitle>
                    </div>
                </CardHeader>
                <CardContent className="space-y-4">
                    {(["credit", "points", "prepaid"] as CurrencyType[]).map((currency) => (
                        <div key={currency} className="flex items-center justify-between">
                            <div>
                                <p className="text-sm text-muted-foreground">{currencyLabels[currency]}</p>
                                <p className="text-lg font-semibold">{wallet.attributes[currency].toLocaleString()}</p>
                            </div>
                            <Button
                                variant="outline"
                                size="sm"
                                onClick={() => openAddDialog(currency)}
                            >
                                <Plus className="h-4 w-4 mr-1" />
                                Add
                            </Button>
                        </div>
                    ))}
                </CardContent>
            </Card>

            <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Add {currencyLabels[activeCurrency]}</DialogTitle>
                        <DialogDescription>
                            Current balance: {wallet.attributes[activeCurrency].toLocaleString()}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="py-4">
                        <Label htmlFor="amount">Amount to add</Label>
                        <Input
                            id="amount"
                            type="number"
                            min="1"
                            placeholder="Enter amount"
                            value={amount}
                            onChange={(e) => setAmount(e.target.value)}
                            onKeyDown={(e) => {
                                if (e.key === "Enter") handleAddBalance();
                            }}
                        />
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDialogOpen(false)}>
                            Cancel
                        </Button>
                        <Button onClick={handleAddBalance} disabled={updateWallet.isPending}>
                            {updateWallet.isPending ? "Adding..." : "Add Balance"}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </>
    );
}
