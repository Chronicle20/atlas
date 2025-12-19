"use client"

import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { MoreHorizontal, Plus, Trash2 } from "lucide-react";

interface AdvancedNpcActionsProps {
  onCreateShop: () => void;
  onDeleteAllShops: () => void;
}

export function AdvancedNpcActions({ onCreateShop, onDeleteAllShops }: AdvancedNpcActionsProps) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm">
          <MoreHorizontal className="h-4 w-4 mr-2" />
          Actions
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={onCreateShop}>
          <Plus className="h-4 w-4 mr-2" />
          Create Shop
        </DropdownMenuItem>
        <DropdownMenuItem onClick={onDeleteAllShops}>
          <Trash2 className="h-4 w-4 mr-2" />
          Delete All Shops
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}