"use client"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { MoreHorizontal, ShoppingBag, MessageCircle } from "lucide-react";
import Link from "next/link";
import { NPC } from "@/types/models/npc";
import { NpcImage } from "./NpcImage";

export interface DropdownAction {
  label: string;
  icon: React.ReactNode;
  onClick: () => void;
}

interface NpcCardProps {
  npc: NPC;
  onShopToggle?: (npcId: number) => void;
  onConversationToggle?: (npcId: number) => void;
  dropdownActions?: DropdownAction[];
}

export function NpcCard({ 
  npc, 
  dropdownActions = [] 
}: NpcCardProps) {
  return (
    <Card className="overflow-hidden">
      <CardHeader className="pb-2 flex justify-between items-start">
        <div className="flex items-center space-x-3 min-w-0 flex-1">
          <NpcImage 
            npcId={npc.id} 
            {...(npc.name && { name: npc.name })}
            {...(npc.iconUrl && { iconUrl: npc.iconUrl })}
            className="w-12 h-12 rounded-md bg-muted flex-shrink-0" 
          />
          <div className="min-w-0 flex-1">
            <CardTitle className="text-lg truncate">
              {npc.name || `NPC #${npc.id}`}
            </CardTitle>
            {npc.name && (
              <p className="text-sm text-muted-foreground">
                ID: {npc.id}
              </p>
            )}
          </div>
        </div>
        
        {/* Conditional dropdown menu - only show if actions are available */}
        {dropdownActions.length > 0 && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="h-8 w-8 p-0 flex-shrink-0">
                <span className="sr-only">Open menu</span>
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {dropdownActions.map((action, index) => (
                <DropdownMenuItem key={index} onClick={action.onClick}>
                  {action.icon}
                  {action.label}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </CardHeader>
      
      <CardContent className="pb-2">
        <div className="flex space-x-2">
          {/* Shop Button */}
          {npc.hasShop ? (
            <Button 
              variant="default" 
              size="sm"
              className="cursor-pointer"
              asChild
              title="Shop Active"
            >
              <Link href={`/npcs/${npc.id}/shop`}>
                <ShoppingBag className="h-4 w-4" />
              </Link>
            </Button>
          ) : (
            <Button 
              variant="outline" 
              size="sm"
              className="cursor-not-allowed opacity-50"
              disabled
              title="Shop Inactive"
            >
              <ShoppingBag className="h-4 w-4" />
            </Button>
          )}

          {/* Conversation Button */}
          {npc.hasConversation ? (
            <Button 
              variant="default" 
              size="sm"
              className="cursor-pointer"
              asChild
              title="Conversation Available"
            >
              <Link href={`/npcs/${npc.id}/conversations`}>
                <MessageCircle className="h-4 w-4" />
              </Link>
            </Button>
          ) : (
            <Button 
              variant="outline" 
              size="sm"
              className="cursor-not-allowed opacity-50"
              disabled
              title="No Conversation"
            >
              <MessageCircle className="h-4 w-4" />
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
}