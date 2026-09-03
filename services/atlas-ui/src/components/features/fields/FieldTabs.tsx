import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

interface FieldTabsProps {
  characterCount: number;
  monsterCount: number;
  objectCount: number;
  tab: string;
  onTabChange: (tab: string) => void;
}

/**
 * FR-21: the field-detail tab shell. Controlled by `?tab=` (owned by the
 * page, D-carried from Task 16's rules-of-hooks fix — no `key` remount).
 * The panel bodies arrive in Tasks 18 (Characters), 19 (Monsters), and 20
 * (Map Objects); this task only builds the shell each will slot into.
 */
export function FieldTabs({
  characterCount,
  monsterCount,
  objectCount,
  tab,
  onTabChange,
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

      <TabsContent value="characters">
        <Card>
          <CardContent className="pt-6">
            <p className="text-sm text-muted-foreground">
              Character roster arrives in a follow-up task.
            </p>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="monsters">
        <Card>
          <CardContent className="pt-6">
            <p className="text-sm text-muted-foreground">
              Live monster table arrives in a follow-up task.
            </p>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="objects">
        <Card>
          <CardContent className="pt-6">
            <p className="text-sm text-muted-foreground">
              Map object table arrives in a follow-up task.
            </p>
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  );
}
