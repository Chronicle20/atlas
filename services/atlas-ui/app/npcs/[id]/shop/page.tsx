"use client"

import { useTenant } from "@/context/tenant-context";
import { useEffect, useState } from "react";
import { createCommodity, deleteCommodity, fetchNPCShop, updateCommodity, updateShop, deleteAllCommoditiesForNPC } from "@/lib/npcs";
import { Commodity, CommodityAttributes, Shop } from "@/types/models/npc";
import { DataTable } from "@/components/data-table";
import {hiddenColumns} from "@/app/npcs/[id]/shop/columns";
import { getColumns } from "./columns";
import { useParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { PlusCircle, Upload, Download, Trash2 } from "lucide-react";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import {toast} from "sonner";

export default function Page() {
    const { activeTenant } = useTenant();
    const params = useParams();
    const npcId = Number(params.id);

    const [, setShop] = useState<Shop | null>(null);
    const [commodities, setCommodities] = useState<Commodity[]>([]);
    const [recharger, setRecharger] = useState<boolean>(false);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    // Dialog states
    const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
    const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
    const [isBulkUpdateDialogOpen, setIsBulkUpdateDialogOpen] = useState(false);
    const [isDeleteAllCommoditiesDialogOpen, setIsDeleteAllCommoditiesDialogOpen] = useState(false);
    const [bulkUpdateJson, setBulkUpdateJson] = useState("");
    const [currentCommodity, setCurrentCommodity] = useState<Commodity | null>(null);
    const [formData, setFormData] = useState<CommodityAttributes>({
        templateId: 0,
        mesoPrice: 0,
        discountRate: 0,
        tokenTemplateId: 0,
        tokenPrice: 0,
        period: 0,
        levelLimit: 0
    });

    const fetchDataAgain = () => {
        if (!activeTenant) return;

        setLoading(true);

        fetchNPCShop(activeTenant, npcId)
            .then((response) => {
                setShop(response.data);
                // Set recharger state from shop data
                setRecharger(response.data.attributes.recharger || false);
                // Extract commodities from included array if available
                if (response.included && response.included.length > 0) {
                    setCommodities(response.included);
                } else {
                    setCommodities([]);
                }
            })
            .catch((err) => setError(err.message))
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        fetchDataAgain();
    }, [activeTenant, npcId]);

    // Reset form data when dialogs are closed
    useEffect(() => {
        if (!isEditDialogOpen) {
            setFormData({
                templateId: 0,
                mesoPrice: 0,
                discountRate: 0,
                tokenTemplateId: 0,
                tokenPrice: 0,
                period: 0,
                levelLimit: 0
            });
        }
    }, [isEditDialogOpen]);

    // Reset form data when create dialog is closed
    useEffect(() => {
        if (!isCreateDialogOpen) {
            setFormData({
                templateId: 0,
                mesoPrice: 0,
                discountRate: 0,
                tokenTemplateId: 0,
                tokenPrice: 0,
                period: 0,
                levelLimit: 0
            });
        }
    }, [isCreateDialogOpen]);

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const { name, value } = e.target;
        setFormData({
            ...formData,
            [name]: Number(value)
        });
    };

    const handleCreateCommodity = async () => {
        if (!activeTenant) return;

        // Validate that either mesoPrice or tokenPrice is greater than zero
        if (formData.mesoPrice <= 0 && formData.tokenPrice <= 0) {
            toast.error("Either Meso Price or Token Price must be greater than zero");
            return;
        }

        // Validate that templateId is greater than zero
        if (formData.templateId <= 0) {
            toast.error("Template ID must be greater than zero");
            return;
        }

        try {
            await createCommodity(activeTenant, npcId, formData);
            setIsCreateDialogOpen(false);
            setFormData({
                templateId: 0,
                mesoPrice: 0,
                discountRate: 0,
                tokenTemplateId: 0,
                tokenPrice: 0,
                period: 0,
                levelLimit: 0
            });
            fetchDataAgain();
            toast.success("Commodity created successfully");
        } catch {
            toast.error("Failed to create commodity");
        }
    };

    const handleEditCommodity = (commodity: Commodity) => {
        setCurrentCommodity(commodity);
        setFormData({
            templateId: commodity.attributes.templateId,
            mesoPrice: commodity.attributes.mesoPrice,
            discountRate: commodity.attributes.discountRate,
            tokenTemplateId: commodity.attributes.tokenTemplateId,
            tokenPrice: commodity.attributes.tokenPrice,
            period: commodity.attributes.period,
            levelLimit: commodity.attributes.levelLimit
        });
        setIsEditDialogOpen(true);
    };

    const handleUpdateCommodity = async () => {
        if (!activeTenant || !currentCommodity) return;

        // Validate that either mesoPrice or tokenPrice is greater than zero
        if (formData.mesoPrice <= 0 && formData.tokenPrice <= 0) {
            toast.error("Either Meso Price or Token Price must be greater than zero");
            return;
        }

        // Validate that templateId is greater than zero
        if (formData.templateId <= 0) {
            toast.error("Template ID must be greater than zero");
            return;
        }

        try {
            await updateCommodity(activeTenant, npcId, currentCommodity.id, formData);
            setIsEditDialogOpen(false);
            setCurrentCommodity(null);
            fetchDataAgain();
            toast.success("Commodity updated successfully");
        } catch {
            toast.error("Failed to update commodity");
        }
    };

    const handleDeleteCommodity = async (commodityId: string) => {
        if (!activeTenant) return;

        try {
            await deleteCommodity(activeTenant, npcId, commodityId);
            fetchDataAgain();
            toast.success("Commodity deleted successfully");
        } catch {
            toast.error("Failed to delete commodity");
        }
    };

    const handleBulkUpdateShop = async () => {
        if (!activeTenant) return;

        try {
            const jsonData = JSON.parse(bulkUpdateJson);

            // Extract commodities from the included array if available
            let commoditiesToUpdate: Commodity[] = [];

            // Check for commodities in the root level included array (standard JSON:API format)
            if (jsonData.included && jsonData.included.length > 0) {
                commoditiesToUpdate = jsonData.included;
            }
            // Fallback to check for commodities in data.included (alternative format)
            else if (jsonData.data.included && jsonData.data.included.length > 0) {
                commoditiesToUpdate = jsonData.data.included;
            }

            // Get recharger value from JSON data if available, otherwise use current state
            const rechargerValue = jsonData.data.attributes?.recharger !== undefined 
                ? jsonData.data.attributes.recharger 
                : recharger;

            await updateShop(activeTenant, npcId, commoditiesToUpdate, rechargerValue);
            setIsBulkUpdateDialogOpen(false);
            setBulkUpdateJson("");
            fetchDataAgain();
            toast.success("Shop updated successfully");
        } catch (err) {
            toast.error("Failed to update shop: " + (err instanceof Error ? err.message : String(err)));
        }
    };

    const handleRechargerToggle = async (checked: boolean) => {
        if (!activeTenant) return;

        try {
            setRecharger(checked);
            await updateShop(activeTenant, npcId, commodities, checked);
            toast.success("Shop recharger status updated successfully");
        } catch (err) {
            // Revert state if update fails
            setRecharger(!checked);
            toast.error("Failed to update shop recharger status: " + (err instanceof Error ? err.message : String(err)));
        }
    };

    const handleExportShop = () => {
        // Create commodity references for relationships section
        const commodityReferences = commodities.map(commodity => ({
            type: "commodities",
            id: commodity.id
        }));

        // Create a JSON object with the shop data in the new format
        const shopData = {
            data: {
                type: "shops",
                id: `shop-${npcId}`,
                attributes: {
                    npcId: npcId,
                    recharger: recharger
                },
                relationships: {
                    commodities: {
                        data: commodityReferences
                    }
                }
            },
            included: commodities
        };

        // Convert to JSON string with pretty formatting
        const jsonString = JSON.stringify(shopData, null, 2);

        // Create a blob with the JSON data
        const blob = new Blob([jsonString], { type: 'application/json' });

        // Create a URL for the blob
        const url = URL.createObjectURL(blob);

        // Create a temporary anchor element
        const a = document.createElement('a');
        a.href = url;
        a.download = `shop-${npcId}.json`;

        // Trigger the download
        document.body.appendChild(a);
        a.click();

        // Clean up
        document.body.removeChild(a);
        URL.revokeObjectURL(url);

        toast.success("Shop data exported successfully");
    };

    const handleDeleteAllCommodities = async () => {
        if (!activeTenant) return;

        try {
            await deleteAllCommoditiesForNPC(activeTenant, npcId);
            toast.success("All commodities deleted successfully");
            setIsDeleteAllCommoditiesDialogOpen(false);
            fetchDataAgain();
        } catch (err) {
            toast.error("Failed to delete all commodities: " + (err instanceof Error ? err.message : String(err)));
        }
    };

    if (loading) return <div>Loading...</div>;
    if (error) return <div>Error: {error}</div>;

    const columns = getColumns({
        npcId,
        onEdit: handleEditCommodity,
        onDelete: handleDeleteCommodity
    });

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
            <div className="flex items-center justify-between">
                <div>
                    <h2 className="text-2xl font-bold tracking-tight">NPC #{npcId} Shop</h2>
                </div>
                <Dialog open={isCreateDialogOpen} onOpenChange={setIsCreateDialogOpen}>
                    <DialogContent className="sm:max-w-xl">
                        <DialogHeader>
                            <DialogTitle>Add New Commodity</DialogTitle>
                        </DialogHeader>
                        <div className="grid gap-4 py-4">
                            <div className="grid grid-cols-4 items-center gap-4">
                                <Label htmlFor="templateId" className="text-right">Template ID</Label>
                                <Input
                                    id="templateId"
                                    name="templateId"
                                    type="number"
                                    value={formData.templateId}
                                    onChange={handleInputChange}
                                    className="col-span-3"
                                />
                            </div>
                            <div className="grid grid-cols-4 items-center gap-4">
                                <Label htmlFor="mesoPrice" className="text-right">Meso Price</Label>
                                <Input
                                    id="mesoPrice"
                                    name="mesoPrice"
                                    type="number"
                                    value={formData.mesoPrice}
                                    onChange={handleInputChange}
                                    className="col-span-3"
                                />
                            </div>
                            <div className="grid grid-cols-4 items-center gap-4">
                                <Label htmlFor="discountRate" className="text-right">Discount Rate</Label>
                                <Input
                                    id="discountRate"
                                    name="discountRate"
                                    type="number"
                                    value={formData.discountRate}
                                    onChange={handleInputChange}
                                    className="col-span-3"
                                />
                            </div>
                            <div className="grid grid-cols-4 items-center gap-4">
                                <Label htmlFor="tokenTemplateId" className="text-right">Token Template ID</Label>
                                <Input
                                    id="tokenTemplateId"
                                    name="tokenTemplateId"
                                    type="number"
                                    value={formData.tokenTemplateId}
                                    onChange={handleInputChange}
                                    className="col-span-3"
                                />
                            </div>
                            <div className="grid grid-cols-4 items-center gap-4">
                                <Label htmlFor="tokenPrice" className="text-right">Token Price</Label>
                                <Input
                                    id="tokenPrice"
                                    name="tokenPrice"
                                    type="number"
                                    value={formData.tokenPrice}
                                    onChange={handleInputChange}
                                    className="col-span-3"
                                />
                            </div>
                            <div className="grid grid-cols-4 items-center gap-4">
                                <Label htmlFor="period" className="text-right">Period</Label>
                                <Input
                                    id="period"
                                    name="period"
                                    type="number"
                                    value={formData.period}
                                    onChange={handleInputChange}
                                    className="col-span-3"
                                />
                            </div>
                            <div className="grid grid-cols-4 items-center gap-4">
                                <Label htmlFor="levelLimit" className="text-right">Level Limit</Label>
                                <Input
                                    id="levelLimit"
                                    name="levelLimit"
                                    type="number"
                                    value={formData.levelLimit}
                                    onChange={handleInputChange}
                                    className="col-span-3"
                                />
                            </div>
                        </div>
                        <DialogFooter>
                            <Button onClick={handleCreateCommodity}>Create</Button>
                        </DialogFooter>
                    </DialogContent>
                </Dialog>
            </div>

            <Dialog open={isEditDialogOpen} onOpenChange={setIsEditDialogOpen}>
                <DialogContent className="sm:max-w-xl">
                    <DialogHeader>
                        <DialogTitle>Edit Commodity</DialogTitle>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid grid-cols-4 items-center gap-4">
                            <Label htmlFor="edit-templateId" className="text-right">Template ID</Label>
                            <Input
                                id="edit-templateId"
                                name="templateId"
                                type="number"
                                value={formData.templateId}
                                onChange={handleInputChange}
                                className="col-span-3"
                            />
                        </div>
                        <div className="grid grid-cols-4 items-center gap-4">
                            <Label htmlFor="edit-mesoPrice" className="text-right">Meso Price</Label>
                            <Input
                                id="edit-mesoPrice"
                                name="mesoPrice"
                                type="number"
                                value={formData.mesoPrice}
                                onChange={handleInputChange}
                                className="col-span-3"
                            />
                        </div>
                        <div className="grid grid-cols-4 items-center gap-4">
                            <Label htmlFor="edit-discountRate" className="text-right">Discount Rate</Label>
                            <Input
                                id="edit-discountRate"
                                name="discountRate"
                                type="number"
                                value={formData.discountRate}
                                onChange={handleInputChange}
                                className="col-span-3"
                            />
                        </div>
                        <div className="grid grid-cols-4 items-center gap-4">
                            <Label htmlFor="edit-tokenTemplateId" className="text-right">Token Template ID</Label>
                            <Input
                                id="edit-tokenTemplateId"
                                name="tokenTemplateId"
                                type="number"
                                value={formData.tokenTemplateId}
                                onChange={handleInputChange}
                                className="col-span-3"
                            />
                        </div>
                        <div className="grid grid-cols-4 items-center gap-4">
                            <Label htmlFor="edit-tokenPrice" className="text-right">Token Price</Label>
                            <Input
                                id="edit-tokenPrice"
                                name="tokenPrice"
                                type="number"
                                value={formData.tokenPrice}
                                onChange={handleInputChange}
                                className="col-span-3"
                            />
                        </div>
                        <div className="grid grid-cols-4 items-center gap-4">
                            <Label htmlFor="edit-period" className="text-right">Period</Label>
                            <Input
                                id="edit-period"
                                name="period"
                                type="number"
                                value={formData.period}
                                onChange={handleInputChange}
                                className="col-span-3"
                            />
                        </div>
                        <div className="grid grid-cols-4 items-center gap-4">
                            <Label htmlFor="edit-levelLimit" className="text-right">Level Limit</Label>
                            <Input
                                id="edit-levelLimit"
                                name="levelLimit"
                                type="number"
                                value={formData.levelLimit}
                                onChange={handleInputChange}
                                className="col-span-3"
                            />
                        </div>
                    </div>
                    <DialogFooter>
                        <Button onClick={handleUpdateCommodity}>Update</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <Dialog open={isBulkUpdateDialogOpen} onOpenChange={setIsBulkUpdateDialogOpen}>
                <DialogContent className="sm:max-w-[600px]">
                    <DialogHeader>
                        <DialogTitle>Bulk Update Shop</DialogTitle>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <Textarea
                            placeholder="Paste JSON data here..."
                            value={bulkUpdateJson}
                            onChange={(e) => setBulkUpdateJson(e.target.value)}
                            className="min-h-[300px] font-mono"
                        />
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setIsBulkUpdateDialogOpen(false)}>
                            Cancel
                        </Button>
                        <Button onClick={handleBulkUpdateShop}>
                            Update Shop
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <Dialog open={isDeleteAllCommoditiesDialogOpen} onOpenChange={setIsDeleteAllCommoditiesDialogOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Delete All Commodities</DialogTitle>
                    </DialogHeader>
                    <div className="py-4">
                        <p className="text-destructive font-semibold">Warning: This action cannot be undone.</p>
                        <p>Are you sure you want to delete all commodities for this shop?</p>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setIsDeleteAllCommoditiesDialogOpen(false)}>
                            Cancel
                        </Button>
                        <Button variant="destructive" onClick={handleDeleteAllCommodities}>
                            Delete All Commodities
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <div className="flex items-center space-x-2 mt-4 mb-4">
                <div className="flex items-center space-x-2">
                    <Switch
                        id="recharger-toggle"
                        checked={recharger}
                        onCheckedChange={handleRechargerToggle}
                    />
                    <Label htmlFor="recharger-toggle" className="font-medium">Recharger</Label>
                </div>
            </div>

            <div className="mt-4">
                <DataTable
                    columns={columns}
                    data={commodities.filter(commodity => {
                        // Filter out commodities that both don't have a meso price and don't have a token price if recharger is set
                        if (recharger && 
                            commodity.attributes.mesoPrice <= 0 && 
                            commodity.attributes.tokenPrice <= 0) {
                            return false;
                        }
                        return true;
                    })}
                    onRefresh={fetchDataAgain}
                    headerActions={[
                        {
                            icon: <PlusCircle className="h-4 w-4" />,
                            label: "Add Commodity",
                            onClick: () => setIsCreateDialogOpen(true)
                        },
                        {
                            icon: <Upload className="h-4 w-4" />,
                            label: "Bulk Update Shop",
                            onClick: () => setIsBulkUpdateDialogOpen(true)
                        },
                        {
                            icon: <Download className="h-4 w-4" />,
                            label: "Export Shop",
                            onClick: handleExportShop
                        },
                        {
                            icon: <Trash2 className="h-4 w-4" />,
                            label: "Delete All Commodities",
                            onClick: () => setIsDeleteAllCommoditiesDialogOpen(true)
                        }
                    ]}
                    initialVisibilityState={hiddenColumns}
                />
            </div>
        </div>
    );
}
