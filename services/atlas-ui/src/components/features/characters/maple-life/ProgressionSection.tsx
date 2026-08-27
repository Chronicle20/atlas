import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useSyncedNumberInput } from "../presets/useSyncedNumberInput";
import type {
  MapleLifeClassDraft,
  ScalarKey,
  StatKey,
} from "./mapleLifeEditorState";

interface ProgressionSectionProps {
  draft: MapleLifeClassDraft;
  onSetStat: (stat: StatKey, value: number) => void;
  onSetScalar: (field: ScalarKey, value: number) => void;
  onSetSpBook: (index: number, value: number) => void;
  /** Blocking messages keyed "sp" | "ap" | "meso" | a stat name. */
  errors?: Record<string, string[]>;
}

const STATS_LABEL =
  "AP already spent to meet the job requirement. HP and MP here EXCLUDE the SP skill's own 29 × effectX contribution, which factory/maple_life.go adds at creation time.";
const UNSPENT_LABEL =
  "AP and SP are what remains UNSPENT at the configured level.";
const BOOK_ZERO_LABEL = "Book 0 is the only book Maple Life reads or spends.";

const STATS = [
  "str",
  "dex",
  "int",
  "luk",
  "hp",
  "mp",
] as const satisfies readonly StatKey[];

function FieldErrors({ messages }: { messages: string[] | undefined }) {
  if (!messages || messages.length === 0) return null;
  return (
    <div className="space-y-0.5">
      {messages.map((message) => (
        <p key={message} className="text-xs text-destructive">
          {message}
        </p>
      ))}
    </div>
  );
}

/**
 * Stats, unspent AP/meso, and the ten SP books for the selected class row
 * (FR-7). Every number input goes through useSyncedNumberInput so a
 * mid-edit keystroke isn't clobbered by the reducer round-trip.
 */
export function ProgressionSection({
  draft,
  onSetStat,
  onSetScalar,
  onSetSpBook,
  errors,
}: ProgressionSectionProps) {
  // Six stat inputs, unrolled (not looped) so each call site is a stable,
  // unconditional hook call — see presets/BaseStatsSection.tsx.
  const [strInput, setStrInput] = useSyncedNumberInput(draft.stats.str);
  const [dexInput, setDexInput] = useSyncedNumberInput(draft.stats.dex);
  const [intInput, setIntInput] = useSyncedNumberInput(draft.stats.int);
  const [lukInput, setLukInput] = useSyncedNumberInput(draft.stats.luk);
  const [hpInput, setHpInput] = useSyncedNumberInput(draft.stats.hp);
  const [mpInput, setMpInput] = useSyncedNumberInput(draft.stats.mp);

  const draftByStat: Record<StatKey, [string, (v: string) => void]> = {
    str: [strInput, setStrInput],
    dex: [dexInput, setDexInput],
    int: [intInput, setIntInput],
    luk: [lukInput, setLukInput],
    hp: [hpInput, setHpInput],
    mp: [mpInput, setMpInput],
  };

  const [apInput, setApInput] = useSyncedNumberInput(draft.ap);
  const [mesoInput, setMesoInput] = useSyncedNumberInput(draft.meso);

  const isBookPoolParsed = draft.spBooks.length === 10;

  // Ten book inputs, likewise unrolled.
  const [book0Input, setBook0Input] = useSyncedNumberInput(
    draft.spBooks[0] ?? 0,
  );
  const [book1Input, setBook1Input] = useSyncedNumberInput(
    draft.spBooks[1] ?? 0,
  );
  const [book2Input, setBook2Input] = useSyncedNumberInput(
    draft.spBooks[2] ?? 0,
  );
  const [book3Input, setBook3Input] = useSyncedNumberInput(
    draft.spBooks[3] ?? 0,
  );
  const [book4Input, setBook4Input] = useSyncedNumberInput(
    draft.spBooks[4] ?? 0,
  );
  const [book5Input, setBook5Input] = useSyncedNumberInput(
    draft.spBooks[5] ?? 0,
  );
  const [book6Input, setBook6Input] = useSyncedNumberInput(
    draft.spBooks[6] ?? 0,
  );
  const [book7Input, setBook7Input] = useSyncedNumberInput(
    draft.spBooks[7] ?? 0,
  );
  const [book8Input, setBook8Input] = useSyncedNumberInput(
    draft.spBooks[8] ?? 0,
  );
  const [book9Input, setBook9Input] = useSyncedNumberInput(
    draft.spBooks[9] ?? 0,
  );

  const bookInputs: [string, (v: string) => void][] = [
    [book0Input, setBook0Input],
    [book1Input, setBook1Input],
    [book2Input, setBook2Input],
    [book3Input, setBook3Input],
    [book4Input, setBook4Input],
    [book5Input, setBook5Input],
    [book6Input, setBook6Input],
    [book7Input, setBook7Input],
    [book8Input, setBook8Input],
    [book9Input, setBook9Input],
  ];

  return (
    <section className="space-y-6">
      <div className="space-y-3">
        <div className="grid gap-3 sm:grid-cols-3">
          {STATS.map((stat) => {
            const [draftValue, setDraftValue] = draftByStat[stat];
            return (
              <div key={stat} className="space-y-1">
                <Label htmlFor={`maple-life-stat-${stat}`}>
                  {stat.toUpperCase()}
                </Label>
                <Input
                  id={`maple-life-stat-${stat}`}
                  aria-label={stat.toUpperCase()}
                  type="number"
                  value={draftValue}
                  onChange={(e) => {
                    setDraftValue(e.target.value);
                    onSetStat(stat, Number(e.target.value));
                  }}
                />
                <FieldErrors messages={errors?.[stat]} />
              </div>
            );
          })}
        </div>
        <p className="text-xs text-muted-foreground">{STATS_LABEL}</p>
      </div>

      <div className="space-y-3">
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-1">
            <Label htmlFor="maple-life-ap">AP</Label>
            <Input
              id="maple-life-ap"
              aria-label="AP"
              type="number"
              value={apInput}
              onChange={(e) => {
                setApInput(e.target.value);
                onSetScalar("ap", Number(e.target.value));
              }}
            />
            <FieldErrors messages={errors?.["ap"]} />
          </div>
          <div className="space-y-1">
            <Label htmlFor="maple-life-meso">Meso</Label>
            <Input
              id="maple-life-meso"
              aria-label="Meso"
              type="number"
              value={mesoInput}
              onChange={(e) => {
                setMesoInput(e.target.value);
                onSetScalar("meso", Number(e.target.value));
              }}
            />
            <FieldErrors messages={errors?.["meso"]} />
          </div>
        </div>
        <p className="text-xs text-muted-foreground">{UNSPENT_LABEL}</p>
      </div>

      <div className="space-y-3">
        <h3 className="text-sm font-semibold">SP books</h3>
        {!isBookPoolParsed && (
          <p className="text-xs text-muted-foreground">
            {`Unparseable SP pool, preserved as loaded: ${draft.spRaw}`}
          </p>
        )}
        <div className="grid gap-3 sm:grid-cols-5">
          {bookInputs.map(([draftValue, setDraftValue], index) => (
            <div key={index} className="space-y-1">
              <Label
                htmlFor={`maple-life-sp-book-${index}`}
                className={index === 0 ? "font-semibold" : undefined}
              >
                {`Book ${index}`}
              </Label>
              <Input
                id={`maple-life-sp-book-${index}`}
                aria-label={`Book ${index}`}
                type="number"
                disabled={!isBookPoolParsed}
                value={draftValue}
                onChange={(e) => {
                  setDraftValue(e.target.value);
                  onSetSpBook(index, Number(e.target.value));
                }}
              />
              {index === 0 && (
                <p className="text-xs text-muted-foreground">
                  {BOOK_ZERO_LABEL}
                </p>
              )}
            </div>
          ))}
        </div>
        <FieldErrors messages={errors?.["sp"]} />
      </div>
    </section>
  );
}
