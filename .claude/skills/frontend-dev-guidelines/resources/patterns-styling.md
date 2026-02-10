# Styling & Theming Patterns

## Overview

Atlas UI uses **Tailwind CSS v4** with **shadcn/ui** components, **tailwind-nord** color palette, and CSS variable-based theming for light/dark mode support via `next-themes`.

## cn() Utility

All conditional classnames go through `cn()` — a combination of `clsx` and `tailwind-merge`:

```typescript
// lib/utils.ts
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}
```

**Usage:**
```tsx
<div className={cn(
  "flex items-center gap-2 p-4",           // Base classes
  variant === "destructive" && "bg-destructive text-destructive-foreground",
  disabled && "opacity-50 cursor-not-allowed",
  className,                                // Allow parent override
)} />
```

**Never** concatenate class strings manually — always use `cn()`.

## Tailwind Class Order

Follow this ordering convention for readability:

```
1. Layout       (flex, grid, inline-flex)
2. Positioning  (relative, absolute, sticky)
3. Box model    (p-4, m-2, w-full, h-10)
4. Typography   (text-sm, font-medium, text-muted-foreground)
5. Visual       (bg-background, border, rounded-lg, shadow-sm)
6. Effects      (transition-colors, animate-spin)
7. States       (hover:bg-accent, focus:ring-2, disabled:opacity-50)
```

## CSS Variable Theming

Theme tokens are defined as CSS variables in `app/globals.css`:

```css
@layer base {
  :root {
    --background: 0 0% 100%;
    --foreground: 222.2 84% 4.9%;
    --card: 0 0% 100%;
    --primary: 222.2 47.4% 11.2%;
    --destructive: 0 84.2% 60.2%;
    --muted: 210 40% 96.1%;
    --accent: 210 40% 96.1%;
    --border: 214.3 31.8% 91.4%;
    --ring: 222.2 84% 4.9%;
    /* ... */
  }

  .dark {
    --background: 222.2 84% 4.9%;
    --foreground: 210 40% 98%;
    /* ... */
  }
}
```

**Use semantic color names**, not raw Tailwind colors:

```tsx
// ✅ Good — semantic, theme-aware
<div className="bg-background text-foreground border-border" />
<p className="text-muted-foreground" />
<Button variant="destructive" />

// ❌ Bad — hard-coded colors, ignores theme
<div className="bg-white text-gray-900 border-gray-200" />
<p className="text-gray-500" />
```

## shadcn/ui Configuration

Defined in `components.json`:

```json
{
  "style": "new-york",
  "rsc": true,
  "tsx": true,
  "tailwind": {
    "config": "tailwind.config.ts",
    "css": "app/globals.css",
    "baseColor": "gray",
    "cssVariables": true
  },
  "aliases": {
    "components": "@/components",
    "utils": "@/lib/utils",
    "ui": "@/components/ui",
    "lib": "@/lib",
    "hooks": "@/hooks"
  },
  "iconLibrary": "lucide"
}
```

## Component Variant Pattern (CVA)

shadcn/ui uses `class-variance-authority` for variant systems:

```typescript
import { cva, type VariantProps } from "class-variance-authority";

const buttonVariants = cva(
  "inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground hover:bg-primary/90",
        destructive: "bg-destructive text-destructive-foreground hover:bg-destructive/90",
        outline: "border border-input bg-background hover:bg-accent",
        ghost: "hover:bg-accent hover:text-accent-foreground",
      },
      size: {
        default: "h-10 px-4 py-2",
        sm: "h-9 rounded-md px-3",
        lg: "h-11 rounded-md px-8",
        icon: "h-10 w-10",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);
```

## Common Layout Patterns

### Full-Height Scrollable Content
```tsx
<div className="flex h-[calc(100vh-10rem)] flex-col overflow-auto">
  {/* Content */}
</div>
```

### Card-Based Detail Layout
```tsx
<div className="space-y-6 p-4">
  <Card>
    <CardHeader><CardTitle>Section Title</CardTitle></CardHeader>
    <CardContent>
      <div className="grid grid-cols-2 gap-4">
        <div><Label>Field</Label><p>{value}</p></div>
      </div>
    </CardContent>
  </Card>
</div>
```

### Responsive Grid
```tsx
<div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
  {items.map(item => <Card key={item.id}>...</Card>)}
</div>
```

### Header with Actions
```tsx
<div className="flex items-center justify-between">
  <h1 className="text-2xl font-bold">{title}</h1>
  <div className="flex gap-2">
    <Button variant="outline" onClick={onRefresh}>
      <RefreshCw className="mr-2 h-4 w-4" /> Refresh
    </Button>
    <Button onClick={onCreate}>
      <Plus className="mr-2 h-4 w-4" /> Create
    </Button>
  </div>
</div>
```

## Icon Usage

Use **Lucide React** icons with consistent sizing:

```tsx
import { Plus, Trash2, ArrowLeft, Loader2, MoreHorizontal } from "lucide-react";

// In buttons
<Button><Plus className="mr-2 h-4 w-4" /> Create</Button>

// Standalone
<Trash2 className="h-4 w-4 text-destructive" />

// Loading
<Loader2 className="h-4 w-4 animate-spin" />
```

Standard sizes: `h-4 w-4` (default), `h-5 w-5` (medium), `h-6 w-6` (large).

## Dark Mode

Managed by `next-themes` with `attribute="class"`:

```tsx
// components/theme-toggle.tsx
import { useTheme } from "next-themes";

export function ThemeToggle() {
  const { theme, setTheme } = useTheme();
  return (
    <Button variant="ghost" size="icon" onClick={() => setTheme(theme === "dark" ? "light" : "dark")}>
      <Sun className="h-4 w-4 rotate-0 scale-100 dark:-rotate-90 dark:scale-0" />
      <Moon className="absolute h-4 w-4 rotate-90 scale-0 dark:rotate-0 dark:scale-100" />
    </Button>
  );
}
```

No manual dark mode logic in components — CSS variables handle it automatically.
