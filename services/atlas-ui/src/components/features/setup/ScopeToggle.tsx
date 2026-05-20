import { Button } from '@/components/ui/button';

export type Scope = 'tenant' | 'shared';

interface ScopeToggleProps {
  value: Scope;
  onChange: (s: Scope) => void;
  region: string;
  version: string;
}

export function ScopeToggle({ value, onChange, region, version }: ScopeToggleProps) {
  return (
    <div className="flex flex-col gap-2" data-testid="scope-toggle">
      <div className="flex gap-2" role="radiogroup" aria-label="Data scope">
        <Button
          type="button"
          role="radio"
          aria-checked={value === 'tenant'}
          variant={value === 'tenant' ? 'default' : 'outline'}
          size="sm"
          onClick={() => onChange('tenant')}
        >
          This tenant
        </Button>
        <Button
          type="button"
          role="radio"
          aria-checked={value === 'shared'}
          variant={value === 'shared' ? 'destructive' : 'outline'}
          size="sm"
          onClick={() => onChange('shared')}
        >
          Canonical (shared)
        </Button>
      </div>
      {value === 'shared' && (
        <p className="text-sm text-destructive">
          This will replace the shared canonical baseline for {region} v{version}.
        </p>
      )}
    </div>
  );
}
