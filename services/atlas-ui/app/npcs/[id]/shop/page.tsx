"use client"

import { useTenant } from "@/context/tenant-context";
import { useEffect, useState } from "react";
import { Commodity, Shop, createCommodity, deleteCommodity, fetchNPCShop, updateCommodity } from "@/lib/npcs";
import { DataTable } from "@/components/data-table";
import { getColumns } from "./columns";
import { useParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { PlusCircle } from "lucide-react";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {toast} from "sonner";

export default function Page() {
    const { activeTenant } = useTenant();
    const params = useParams();
    const npcId = Number(params.id);

    const [, setShop] = useState<Shop | null>(null);
    const [commodities, setCommodities] = useState<Commodity[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    // Dialog states
    const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
    const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
    const [currentCommodity, setCurrentCommodity] = useState<Commodity | null>(null);
    const [formData, setFormData] = useState<Partial<Commodity>>({
        templateId: 0,
        mesoPrice: 0,
        discountRate: 0,
        tokenItemId: 0,
        tokenPrice: 0,
        period: 0,
        levelLimit: 0
    });

    const fetchDataAgain = () => {
        if (!activeTenant) return;

        setLoading(true);

        fetchNPCShop(activeTenant, npcId)
            .then((shopData) => {
                setShop(shopData);
                setCommodities(shopData.attributes.commodities);
            })
            .catch((err) => setError(err.message))
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        fetchDataAgain();
    }, [activeTenant, npcId]);

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const { name, value } = e.target;
        setFormData({
            ...formData,
            [name]: Number(value)
        });
    };

    const handleCreateCommodity = async () => {
        if (!activeTenant) return;

        try {
            await createCommodity(activeTenant, npcId, formData as Omit<Commodity, 'id'>);
            setIsCreateDialogOpen(false);
            setFormData({
                templateId: 0,
                mesoPrice: 0,
                discountRate: 0,
                tokenItemId: 0,
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
            templateId: commodity.templateId,
            mesoPrice: commodity.mesoPrice,
            discountRate: commodity.discountRate,
            tokenItemId: commodity.tokenItemId,
            tokenPrice: commodity.tokenPrice,
            period: commodity.period,
            levelLimit: commodity.levelLimit
        });
        setIsEditDialogOpen(true);
    };

    const handleUpdateCommodity = async () => {
        if (!activeTenant || !currentCommodity) return;

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
                    <DialogContent>
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
                                <Label htmlFor="tokenItemId" className="text-right">Token Item ID</Label>
                                <Input
                                    id="tokenItemId"
                                    name="tokenItemId"
                                    type="number"
                                    value={formData.tokenItemId}
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
                <DialogContent>
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
                            <Label htmlFor="edit-tokenItemId" className="text-right">Token Item ID</Label>
                            <Input
                                id="edit-tokenItemId"
                                name="tokenItemId"
                                type="number"
                                value={formData.tokenItemId}
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

            <div className="mt-4">
                <DataTable
                    columns={columns}
                    data={commodities}
                    onRefresh={fetchDataAgain}
                    headerActions={[
                        {
                            icon: <PlusCircle className="h-4 w-4" />,
                            label: "Add Commodity",
                            onClick: () => setIsCreateDialogOpen(true)
                        }
                    ]}
                />
            </div>
        </div>
    );
}
