"use client"

import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import {
    Form,
    FormControl,
    FormDescription,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from "@/components/ui/form";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { toast } from "sonner";
import { bansService } from "@/services/api/bans.service";
import { BanType, BanTypeLabels, BanReasonCode, BanReasonCodeLabels, type CreateBanRequest } from "@/types/models/ban";
import type { Tenant } from "@/types/models/tenant";
import { Loader2 } from "lucide-react";

// IP/CIDR validation regex
const ipRegex = /^(\d{1,3}\.){3}\d{1,3}(\/\d{1,2})?$/;

const formSchema = z.object({
    banType: z.nativeEnum(BanType),
    value: z.string().min(1, "Value is required"),
    reason: z.string().optional(),
    reasonCode: z.nativeEnum(BanReasonCode),
    permanent: z.boolean(),
    expiresAt: z.string().optional(),
    issuedBy: z.string().optional(),
}).refine((data) => {
    // IP validation for IP type bans
    if (data.banType === BanType.IP && !ipRegex.test(data.value)) {
        return false;
    }
    return true;
}, {
    message: "Invalid IP address or CIDR format (e.g., 192.168.1.1 or 192.168.1.0/24)",
    path: ["value"],
}).refine((data) => {
    // Expiration validation for non-permanent bans
    if (!data.permanent && !data.expiresAt) {
        return false;
    }
    return true;
}, {
    message: "Expiration date is required for non-permanent bans",
    path: ["expiresAt"],
}).refine((data) => {
    // Expiration must be in the future
    if (!data.permanent && data.expiresAt) {
        const expiresAt = new Date(data.expiresAt).getTime();
        if (expiresAt <= Date.now()) {
            return false;
        }
    }
    return true;
}, {
    message: "Expiration date must be in the future",
    path: ["expiresAt"],
});

type FormValues = z.infer<typeof formSchema>;

interface CreateBanDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    tenant: Tenant | null;
    onSuccess?: () => void;
    /** Pre-fill values for creating ban from history */
    prefill?: {
        banType?: BanType;
        value?: string;
    };
}

export function CreateBanDialog({ open, onOpenChange, tenant, onSuccess, prefill }: CreateBanDialogProps) {
    const [isSubmitting, setIsSubmitting] = useState(false);

    const form = useForm<FormValues>({
        resolver: zodResolver(formSchema),
        defaultValues: {
            banType: prefill?.banType ?? BanType.IP,
            value: prefill?.value ?? "",
            reason: "",
            reasonCode: BanReasonCode.Unspecified,
            permanent: false,
            expiresAt: "",
            issuedBy: "",
        },
    });

    const isPermanent = form.watch("permanent");
    const banType = form.watch("banType");

    const onSubmit = async (values: FormValues) => {
        if (!tenant) {
            toast.error("No tenant selected");
            return;
        }

        setIsSubmitting(true);

        try {
            const request: CreateBanRequest = {
                banType: values.banType,
                value: values.value,
                reason: values.reason || "",
                reasonCode: values.reasonCode,
                permanent: values.permanent,
                expiresAt: values.permanent ? 0 : new Date(values.expiresAt!).getTime(),
                issuedBy: values.issuedBy || "",
            };

            await bansService.createBan(tenant, request);
            toast.success("Ban created successfully");
            form.reset();
            onOpenChange(false);
            onSuccess?.();
        } catch (error) {
            toast.error("Failed to create ban: " + (error instanceof Error ? error.message : "Unknown error"));
        } finally {
            setIsSubmitting(false);
        }
    };

    const getValuePlaceholder = () => {
        switch (banType) {
            case BanType.IP:
                return "e.g., 192.168.1.1 or 192.168.1.0/24";
            case BanType.HWID:
                return "Hardware ID value";
            case BanType.Account:
                return "Account ID (numeric)";
            default:
                return "Enter value";
        }
    };

    const getValueDescription = () => {
        switch (banType) {
            case BanType.IP:
                return "IP address or CIDR range (e.g., 192.168.1.0/24 for a range)";
            case BanType.HWID:
                return "Hardware ID to ban";
            case BanType.Account:
                return "Account ID number to ban";
            default:
                return "";
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[500px]">
                <DialogHeader>
                    <DialogTitle>Create New Ban</DialogTitle>
                    <DialogDescription>
                        Create a new ban to restrict access. Fill in the details below.
                    </DialogDescription>
                </DialogHeader>

                <Form {...form}>
                    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                        <FormField
                            control={form.control}
                            name="banType"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Ban Type</FormLabel>
                                    <Select
                                        onValueChange={(value) => field.onChange(Number(value))}
                                        defaultValue={field.value.toString()}
                                    >
                                        <FormControl>
                                            <SelectTrigger>
                                                <SelectValue placeholder="Select ban type" />
                                            </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                            {Object.entries(BanTypeLabels).map(([value, label]) => (
                                                <SelectItem key={value} value={value}>
                                                    {label}
                                                </SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="value"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Value</FormLabel>
                                    <FormControl>
                                        <Input placeholder={getValuePlaceholder()} {...field} />
                                    </FormControl>
                                    <FormDescription>{getValueDescription()}</FormDescription>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="reasonCode"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Reason Code</FormLabel>
                                    <Select
                                        onValueChange={(value) => field.onChange(Number(value))}
                                        defaultValue={field.value.toString()}
                                    >
                                        <FormControl>
                                            <SelectTrigger>
                                                <SelectValue placeholder="Select reason" />
                                            </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                            {Object.entries(BanReasonCodeLabels).map(([value, label]) => (
                                                <SelectItem key={value} value={value}>
                                                    {label}
                                                </SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="reason"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Reason (Optional)</FormLabel>
                                    <FormControl>
                                        <Textarea
                                            placeholder="Additional details about this ban..."
                                            className="resize-none"
                                            {...field}
                                        />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="permanent"
                            render={({ field }) => (
                                <FormItem className="flex flex-row items-center justify-between rounded-lg border p-3">
                                    <div className="space-y-0.5">
                                        <FormLabel>Permanent Ban</FormLabel>
                                        <FormDescription>
                                            This ban will never expire
                                        </FormDescription>
                                    </div>
                                    <FormControl>
                                        <Switch
                                            checked={field.value}
                                            onCheckedChange={field.onChange}
                                        />
                                    </FormControl>
                                </FormItem>
                            )}
                        />

                        {!isPermanent && (
                            <FormField
                                control={form.control}
                                name="expiresAt"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Expiration Date</FormLabel>
                                        <FormControl>
                                            <Input
                                                type="datetime-local"
                                                min={new Date().toISOString().slice(0, 16)}
                                                {...field}
                                            />
                                        </FormControl>
                                        <FormDescription>
                                            When this ban should expire
                                        </FormDescription>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                        )}

                        <FormField
                            control={form.control}
                            name="issuedBy"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Issued By (Optional)</FormLabel>
                                    <FormControl>
                                        <Input placeholder="Admin name or identifier" {...field} />
                                    </FormControl>
                                    <FormDescription>
                                        Who is issuing this ban
                                    </FormDescription>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <DialogFooter>
                            <Button
                                type="button"
                                variant="outline"
                                onClick={() => onOpenChange(false)}
                                disabled={isSubmitting}
                            >
                                Cancel
                            </Button>
                            <Button type="submit" disabled={isSubmitting}>
                                {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                                Create Ban
                            </Button>
                        </DialogFooter>
                    </form>
                </Form>
            </DialogContent>
        </Dialog>
    );
}
