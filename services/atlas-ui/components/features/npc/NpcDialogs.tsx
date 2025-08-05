"use client"

import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";

interface NpcDialogsProps {
  isCreateShopDialogOpen: boolean;
  setIsCreateShopDialogOpen: (open: boolean) => void;
  isDeleteAllShopsDialogOpen: boolean;
  setIsDeleteAllShopsDialogOpen: (open: boolean) => void;
  isBulkUpdateShopDialogOpen: boolean;
  setIsBulkUpdateShopDialogOpen: (open: boolean) => void;
  createShopJson: string;
  setCreateShopJson: (json: string) => void;
  bulkUpdateShopJson: string;
  setBulkUpdateShopJson: (json: string) => void;
  handleCreateShop: () => Promise<void>;
  handleDeleteAllShops: () => Promise<void>;
  handleBulkUpdateShop: () => Promise<void>;
}

export function NpcDialogs({
  isCreateShopDialogOpen,
  setIsCreateShopDialogOpen,
  isDeleteAllShopsDialogOpen,
  setIsDeleteAllShopsDialogOpen,
  isBulkUpdateShopDialogOpen,
  setIsBulkUpdateShopDialogOpen,
  createShopJson,
  setCreateShopJson,
  bulkUpdateShopJson,
  setBulkUpdateShopJson,
  handleCreateShop,
  handleDeleteAllShops,
  handleBulkUpdateShop,
}: NpcDialogsProps) {
  return (
    <>
      {/* Create Shop Dialog */}
      <Dialog open={isCreateShopDialogOpen} onOpenChange={setIsCreateShopDialogOpen}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Create Shop</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <Textarea
              placeholder="Paste JSON data here..."
              value={createShopJson}
              onChange={(e) => setCreateShopJson(e.target.value)}
              className="min-h-[300px] font-mono"
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsCreateShopDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleCreateShop}>
              Create Shop
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete All Shops Dialog */}
      <Dialog open={isDeleteAllShopsDialogOpen} onOpenChange={setIsDeleteAllShopsDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete All Shops</DialogTitle>
          </DialogHeader>
          <div className="py-4">
            <p className="text-destructive font-semibold">Warning: This action cannot be undone.</p>
            <p>Are you sure you want to delete all shops for the current tenant?</p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsDeleteAllShopsDialogOpen(false)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleDeleteAllShops}>
              Delete All Shops
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Bulk Update Shop Dialog */}
      <Dialog open={isBulkUpdateShopDialogOpen} onOpenChange={setIsBulkUpdateShopDialogOpen}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Bulk Update Shop</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <Textarea
              placeholder="Paste JSON data here..."
              value={bulkUpdateShopJson}
              onChange={(e) => setBulkUpdateShopJson(e.target.value)}
              className="min-h-[300px] font-mono"
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsBulkUpdateShopDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleBulkUpdateShop}>
              Update Shop
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}