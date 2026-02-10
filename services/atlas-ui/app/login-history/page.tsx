"use client"

import { useTenant } from "@/context/tenant-context";
import { useCallback, useState } from "react";
import { loginHistoryService } from "@/services/api/login-history.service";
import { LoginHistoryEntry, BanType } from "@/types/models/ban";
import { CreateBanDialog } from "@/components/features/bans/CreateBanDialog";
import { Toaster, toast } from "sonner";
import { createErrorFromUnknown } from "@/types/api/errors";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { History, Search, Loader2, MoreHorizontal, Ban, CheckCircle, XCircle } from "lucide-react";

export default function LoginHistoryPage() {
    const { activeTenant } = useTenant();
    const [searchCriteria, setSearchCriteria] = useState({
        ip: "",
        hwid: "",
        accountId: "",
    });
    const [entries, setEntries] = useState<LoginHistoryEntry[]>([]);
    const [loading, setLoading] = useState(false);
    const [hasSearched, setHasSearched] = useState(false);
    const [createDialogOpen, setCreateDialogOpen] = useState(false);
    const [prefillData, setPrefillData] = useState<{ banType: BanType; value: string } | null>(null);

    const handleSearch = useCallback(async () => {
        if (!activeTenant) {
            toast.error("No tenant selected");
            return;
        }

        const { ip, hwid, accountId } = searchCriteria;
        if (!ip && !hwid && !accountId) {
            toast.error("Please enter at least one search criteria");
            return;
        }

        setLoading(true);
        setHasSearched(true);

        try {
            const criteria: { ip?: string; hwid?: string; accountId?: number } = {};
            if (ip) criteria.ip = ip;
            if (hwid) criteria.hwid = hwid;
            if (accountId) criteria.accountId = Number(accountId);
            const data = await loginHistoryService.search(activeTenant, criteria);
            setEntries(data);

            if (data.length === 0) {
                toast.info("No login history found matching your criteria");
            }
        } catch (err: unknown) {
            const errorInfo = createErrorFromUnknown(err, "Failed to search login history");
            toast.error(errorInfo.message);
        } finally {
            setLoading(false);
        }
    }, [activeTenant, searchCriteria]);

    const handleClear = () => {
        setSearchCriteria({ ip: "", hwid: "", accountId: "" });
        setEntries([]);
        setHasSearched(false);
    };

    const handleCreateBan = (entry: LoginHistoryEntry, type: "ip" | "hwid" | "account") => {
        let banType: BanType;
        let value: string;

        switch (type) {
            case "ip":
                banType = BanType.IP;
                value = entry.attributes.ipAddress;
                break;
            case "hwid":
                banType = BanType.HWID;
                value = entry.attributes.hwid;
                break;
            case "account":
                banType = BanType.Account;
                value = entry.attributes.accountId.toString();
                break;
        }

        setPrefillData({ banType, value });
        setCreateDialogOpen(true);
    };

    return (
        <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
            <div className="flex items-center gap-2">
                <History className="h-6 w-6" />
                <h2 className="text-2xl font-bold tracking-tight">Login History</h2>
            </div>

            <Card>
                <CardHeader>
                    <CardTitle>Search Login History</CardTitle>
                    <CardDescription>
                        Search for login attempts by IP address, Hardware ID, or Account ID.
                        Results can be used to identify suspicious activity and create bans.
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <div className="grid gap-4 md:grid-cols-3">
                        <div className="space-y-2">
                            <Label htmlFor="ip">IP Address</Label>
                            <Input
                                id="ip"
                                placeholder="e.g., 192.168.1.1"
                                value={searchCriteria.ip}
                                onChange={(e) => setSearchCriteria(prev => ({ ...prev, ip: e.target.value }))}
                            />
                        </div>
                        <div className="space-y-2">
                            <Label htmlFor="hwid">Hardware ID</Label>
                            <Input
                                id="hwid"
                                placeholder="Hardware ID"
                                value={searchCriteria.hwid}
                                onChange={(e) => setSearchCriteria(prev => ({ ...prev, hwid: e.target.value }))}
                            />
                        </div>
                        <div className="space-y-2">
                            <Label htmlFor="accountId">Account ID</Label>
                            <Input
                                id="accountId"
                                type="number"
                                placeholder="e.g., 12345"
                                value={searchCriteria.accountId}
                                onChange={(e) => setSearchCriteria(prev => ({ ...prev, accountId: e.target.value }))}
                            />
                        </div>
                    </div>
                    <div className="flex gap-2 mt-4">
                        <Button onClick={handleSearch} disabled={loading}>
                            {loading ? (
                                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            ) : (
                                <Search className="mr-2 h-4 w-4" />
                            )}
                            Search
                        </Button>
                        <Button variant="outline" onClick={handleClear} disabled={loading}>
                            Clear
                        </Button>
                    </div>
                </CardContent>
            </Card>

            {hasSearched && (
                <Card className="flex-1 min-h-0 flex flex-col">
                    <CardHeader className="shrink-0">
                        <CardTitle>
                            Results
                            {entries.length > 0 && (
                                <span className="ml-2 text-muted-foreground font-normal">
                                    ({entries.length} {entries.length === 1 ? 'entry' : 'entries'})
                                </span>
                            )}
                        </CardTitle>
                    </CardHeader>
                    <CardContent className="flex-1 min-h-0 flex flex-col">
                        {entries.length === 0 ? (
                            <div className="text-center py-8 text-muted-foreground">
                                No login history found matching your search criteria.
                            </div>
                        ) : (
                            <div className="rounded-md border flex-1 min-h-0 overflow-auto">
                                <Table>
                                    <TableHeader className="sticky top-0 bg-background z-10">
                                        <TableRow>
                                            <TableHead>Account</TableHead>
                                            <TableHead>IP Address</TableHead>
                                            <TableHead>Hardware ID</TableHead>
                                            <TableHead>Status</TableHead>
                                            <TableHead>Failure Reason</TableHead>
                                            <TableHead className="w-[50px]"></TableHead>
                                        </TableRow>
                                    </TableHeader>
                                    <TableBody>
                                        {entries.map((entry) => (
                                            <TableRow key={entry.id}>
                                                <TableCell>
                                                    <div className="flex flex-col">
                                                        <span className="font-medium">{entry.attributes.accountName}</span>
                                                        <span className="text-xs text-muted-foreground">
                                                            ID: {entry.attributes.accountId}
                                                        </span>
                                                    </div>
                                                </TableCell>
                                                <TableCell className="font-mono text-sm">
                                                    {entry.attributes.ipAddress || "-"}
                                                </TableCell>
                                                <TableCell className="font-mono text-sm max-w-[150px] truncate">
                                                    {entry.attributes.hwid || "-"}
                                                </TableCell>
                                                <TableCell>
                                                    {entry.attributes.success ? (
                                                        <Badge variant="secondary" className="bg-green-100 text-green-800">
                                                            <CheckCircle className="mr-1 h-3 w-3" />
                                                            Success
                                                        </Badge>
                                                    ) : (
                                                        <Badge variant="secondary" className="bg-red-100 text-red-800">
                                                            <XCircle className="mr-1 h-3 w-3" />
                                                            Failed
                                                        </Badge>
                                                    )}
                                                </TableCell>
                                                <TableCell className="text-sm text-muted-foreground">
                                                    {entry.attributes.failureReason || "-"}
                                                </TableCell>
                                                <TableCell>
                                                    <DropdownMenu>
                                                        <DropdownMenuTrigger asChild>
                                                            <Button variant="ghost" className="h-8 w-8 p-0">
                                                                <span className="sr-only">Open menu</span>
                                                                <MoreHorizontal className="h-4 w-4" />
                                                            </Button>
                                                        </DropdownMenuTrigger>
                                                        <DropdownMenuContent align="end">
                                                            {entry.attributes.ipAddress && (
                                                                <DropdownMenuItem onClick={() => handleCreateBan(entry, "ip")}>
                                                                    <Ban className="mr-2 h-4 w-4" />
                                                                    Ban IP
                                                                </DropdownMenuItem>
                                                            )}
                                                            {entry.attributes.hwid && (
                                                                <DropdownMenuItem onClick={() => handleCreateBan(entry, "hwid")}>
                                                                    <Ban className="mr-2 h-4 w-4" />
                                                                    Ban HWID
                                                                </DropdownMenuItem>
                                                            )}
                                                            <DropdownMenuItem onClick={() => handleCreateBan(entry, "account")}>
                                                                <Ban className="mr-2 h-4 w-4" />
                                                                Ban Account
                                                            </DropdownMenuItem>
                                                        </DropdownMenuContent>
                                                    </DropdownMenu>
                                                </TableCell>
                                            </TableRow>
                                        ))}
                                    </TableBody>
                                </Table>
                            </div>
                        )}
                    </CardContent>
                </Card>
            )}

            <CreateBanDialog
                open={createDialogOpen}
                onOpenChange={(open) => {
                    setCreateDialogOpen(open);
                    if (!open) setPrefillData(null);
                }}
                tenant={activeTenant}
                {...(prefillData ? { prefill: prefillData } : {})}
                onSuccess={() => {
                    toast.success("Ban created. You can view it in the Bans page.");
                }}
            />

            <Toaster richColors />
        </div>
    );
}
