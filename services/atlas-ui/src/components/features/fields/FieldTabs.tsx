import type { ReactNode } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

interface FieldTabsProps {
  characterCount: number;
  monsterCount: number;
  objectCount: number;
  tab: string;
  onTabChange: (tab: string) => void;
  /** FR-25..27 tab body (task 18). Required — no placeholder left standing. */
  characters: ReactNode;
  /** FR-28..31 tab body (task 19). Required — no placeholder left standing. */
  monsters: ReactNode;
  /** FR-32..33, FR-38 tab body (task 20). Required — no placeholder left standing. */
  objects: ReactNode;
}

/**
 * FR-21: the field-detail tab shell. Controlled by `?tab=` (owned by the
 * page, D-carried from Task 16's rules-of-hooks fix — no `key` remount).
 * Each panel body is a slot the page fills; `characters` landed in Task 18,
 * `monsters` in Task 19, `objects` in Task 20.
 */
export function FieldTabs({
  characterCount,
  monsterCount,
  objectCount,
  tab,
  onTabChange,
  characters,
  monsters,
  objects,
}: FieldTabsProps) {
  return (
    <Tabs value={tab} onValueChange={onTabChange} className="flex flex-col">
      <TabsList>
        <TabsTrigger value="characters">
          {`Characters (${characterCount})`}
        </TabsTrigger>
        <TabsTrigger value="monsters">
          {`Monsters (${monsterCount})`}
        </TabsTrigger>
        <TabsTrigger value="objects">
          {`Map Objects (${objectCount})`}
        </TabsTrigger>
      </TabsList>

      <TabsContent value="characters">{characters}</TabsContent>

      <TabsContent value="monsters">{monsters}</TabsContent>

      <TabsContent value="objects">{objects}</TabsContent>
    </Tabs>
  );
}
