"use client"

import { useState, useRef } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Loader2, Upload, Database, MessageSquare, Store, DoorOpen, Zap, Package, HelpCircle } from "lucide-react";
import { Toaster, toast } from "sonner";
import {
  useSeedDrops,
  useSeedGachapons,
  useSeedNpcConversations,
  useSeedQuestConversations,
  useSeedNpcShops,
  useSeedPortalScripts,
  useSeedReactorScripts,
  useUploadGameData,
} from "@/lib/hooks/api/useSeed";

interface SeedButtonProps {
  label: string;
  description: string;
  icon: React.ReactNode;
  isPending: boolean;
  onClick: () => void;
}

function SeedButton({ label, description, icon, isPending, onClick }: SeedButtonProps) {
  return (
    <Card>
      <CardContent className="flex items-center justify-between p-4">
        <div className="flex items-center gap-3">
          <div className="text-muted-foreground">{icon}</div>
          <div>
            <p className="font-medium text-sm">{label}</p>
            <p className="text-xs text-muted-foreground">{description}</p>
          </div>
        </div>
        <Button size="sm" variant="outline" onClick={onClick} disabled={isPending}>
          {isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : "Seed"}
        </Button>
      </CardContent>
    </Card>
  );
}

export default function SetupPage() {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);

  const seedDrops = useSeedDrops();
  const seedGachapons = useSeedGachapons();
  const seedNpcConversations = useSeedNpcConversations();
  const seedQuestConversations = useSeedQuestConversations();
  const seedNpcShops = useSeedNpcShops();
  const seedPortalScripts = useSeedPortalScripts();
  const seedReactorScripts = useSeedReactorScripts();
  const uploadGameData = useUploadGameData();

  const handleSeed = (mutation: { mutate: () => void; }, label: string) => {
    mutation.mutate();
    toast.info(`Seeding ${label}...`);
  };

  const handleFileUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    if (!file.name.endsWith('.zip')) {
      toast.error("Please select a .zip file");
      return;
    }

    setUploading(true);
    uploadGameData.mutate(file, {
      onSuccess: () => {
        toast.success("Game data uploaded successfully");
        setUploading(false);
      },
      onError: (error) => {
        toast.error(`Upload failed: ${error.message}`);
        setUploading(false);
      },
    });

    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const seedActions = [
    {
      label: "Monster & Reactor Drops",
      description: "Seed drop tables for monsters and reactors",
      icon: <Database className="h-5 w-5" />,
      mutation: seedDrops,
    },
    {
      label: "Gachapons",
      description: "Seed gachapon machine configurations",
      icon: <Package className="h-5 w-5" />,
      mutation: seedGachapons,
    },
    {
      label: "NPC Conversations",
      description: "Seed NPC conversation scripts",
      icon: <MessageSquare className="h-5 w-5" />,
      mutation: seedNpcConversations,
    },
    {
      label: "Quest Conversations",
      description: "Seed quest conversation scripts",
      icon: <HelpCircle className="h-5 w-5" />,
      mutation: seedQuestConversations,
    },
    {
      label: "NPC Shops",
      description: "Seed NPC shop inventories",
      icon: <Store className="h-5 w-5" />,
      mutation: seedNpcShops,
    },
    {
      label: "Portal Scripts",
      description: "Seed portal action scripts",
      icon: <DoorOpen className="h-5 w-5" />,
      mutation: seedPortalScripts,
    },
    {
      label: "Reactor Scripts",
      description: "Seed reactor action scripts",
      icon: <Zap className="h-5 w-5" />,
      mutation: seedReactorScripts,
    },
  ];

  return (
    <div className="flex flex-col space-y-6 p-10 pb-16 overflow-y-auto">
      <div className="items-center justify-between space-y-2">
        <h2 className="text-2xl font-bold tracking-tight">Bootstrap</h2>
        <p className="text-muted-foreground">Upload game data and seed service databases.</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Upload Game Data</CardTitle>
          <CardDescription>
            Upload a WZ data export (.zip) to populate the game data service with monsters, maps, NPCs, items, and more.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-4">
            <input
              ref={fileInputRef}
              type="file"
              accept=".zip"
              className="hidden"
              onChange={handleFileUpload}
            />
            <Button
              onClick={() => fileInputRef.current?.click()}
              disabled={uploading}
            >
              {uploading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Uploading...
                </>
              ) : (
                <>
                  <Upload className="mr-2 h-4 w-4" />
                  Select ZIP File
                </>
              )}
            </Button>
          </div>
        </CardContent>
      </Card>

      <div>
        <h3 className="text-lg font-semibold mb-3">Seed Data</h3>
        <p className="text-sm text-muted-foreground mb-4">
          Populate individual service databases from their configured data sources.
        </p>
        <div className="grid gap-3">
          {seedActions.map((action) => (
            <SeedButton
              key={action.label}
              label={action.label}
              description={action.description}
              icon={action.icon}
              isPending={action.mutation.isPending}
              onClick={() => handleSeed(action.mutation, action.label)}
            />
          ))}
        </div>
      </div>

      <Toaster richColors />
    </div>
  );
}
