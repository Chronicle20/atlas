import type { ReactNode } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

interface FieldTabsProps {
  characterCount: number;
  monsterCount: number;
  objectCount: number;
  tab: string;
  onTabChange: (tab: string) => void;
  /** FR-25..27 tab body (task 18). Required — no placeholder left standing. */
  characters: ReactNode;
  /** Task 19 fills this; falls back to the placeholder until then. */
  monsters?: ReactNode;
  /** Task 20 fills this; falls back to the placeholder until then. */
  objects?: ReactNode;
}

/**
 * FR-21: the field-detail tab shell. Controlled by `?tab=` (owned by the
 * page, D-carried from Task 16's rules-of-hooks fix — no `key` remount).
 * Each panel body is a slot the page fills; `characters` landed in Task 18,
 * `monsters`/`objects` arrive in Tasks 19/20 the same way.
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

      <TabsContent value="monsters">
        {monsters ?? (
          <Card>
            <CardContent className="pt-6">
              <p className="text-sm text-muted-foreground">
                Live monster table arrives in a follow-up task.
              </p>
            </CardContent>
          </Card>
        )}
      </TabsContent>

      <TabsContent value="objects">
        {objects ?? (
          <Card>
            <CardContent className="pt-6">
              <p className="text-sm text-muted-foreground">
                Map object table arrives in a follow-up task.
              </p>
            </CardContent>
          </Card>
        )}
      </TabsContent>
    </Tabs>
  );
}
