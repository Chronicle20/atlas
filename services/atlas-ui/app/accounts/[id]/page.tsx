"use client"

import { useEffect, useState } from "react"
import { useParams } from "next/navigation"
import { Toaster } from "@/components/ui/sonner"
import { useTenant } from "@/context/tenant-context"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { accountsService } from "@/services/api/accounts.service"
import { walletService } from "@/services/api/wallet.service"
import type { Account } from "@/types/models/account"
import type { Wallet } from "@/types/models/wallet"
import { createErrorFromUnknown } from "@/types/api/errors"
import { ErrorDisplay } from "@/components/common"
import { AccountDetailSkeleton } from "@/components/common/skeletons/AccountDetailSkeleton"
import { WalletPanel } from "@/components/features/accounts/WalletPanel"

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

export default function AccountDetailPage() {
    const { id } = useParams()
    const { activeTenant } = useTenant()

    const [account, setAccount] = useState<Account | null>(null)
    const [wallet, setWallet] = useState<Wallet | null>(null)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [walletError, setWalletError] = useState<string | null>(null)

    useEffect(() => {
        if (!activeTenant || !id) return

        setLoading(true)
        setError(null)
        setWalletError(null)

        const accountPromise = accountsService.getAccountById(activeTenant, String(id))
            .then(setAccount)
            .catch((err: unknown) => {
                const errorInfo = createErrorFromUnknown(err, "Failed to fetch account")
                setError(errorInfo.message)
            })

        const walletPromise = walletService.getWallet(activeTenant, String(id))
            .then(setWallet)
            .catch((err: unknown) => {
                setWalletError("Wallet not found for this account")
            })

        Promise.all([accountPromise, walletPromise])
            .finally(() => setLoading(false))
    }, [activeTenant, id])

    if (loading) return <AccountDetailSkeleton />
    if (error || !account) return <ErrorDisplay error={error || "Account not found"} className="p-4" />

    const lastLogin = account.attributes.lastLogin
        ? new Date(account.attributes.lastLogin).toLocaleString()
        : "Never"

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16 h-screen overflow-auto">
            <div className="items-center justify-between space-y-2">
                <div>
                    <h2 className="text-2xl font-bold tracking-tight">{account.attributes.name}</h2>
                </div>
            </div>

            <div className="flex flex-row gap-6">
                {/* Account Info Card */}
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

                {/* Wallet Panel */}
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
    )
}
