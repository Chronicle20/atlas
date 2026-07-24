import { useState, type ReactNode } from "react";
import { X, Plus } from "lucide-react";
import type { CharacterPresetAttributes } from "@/types/models/template";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";

interface IdentitySectionProps {
  attrs: CharacterPresetAttributes;
  onSetField: (
    path: "name" | "defaultName" | "description",
    value: string,
  ) => void;
  onAddTag: (tag: string) => void;
  onRemoveTag: (tag: string) => void;
  /** Rendered top-right of the section header (kebab anchor). */
  actions?: ReactNode;
}

export function IdentitySection({
  attrs,
  onSetField,
  onAddTag,
  onRemoveTag,
  actions,
}: IdentitySectionProps) {
  const [addingTag, setAddingTag] = useState(false);
  const [newTag, setNewTag] = useState("");

  const handleAddTag = () => {
    const trimmed = newTag.trim();
    if (trimmed) {
      onAddTag(trimmed);
    }
    setNewTag("");
    setAddingTag(false);
  };

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold">Identity</h3>
        {actions}
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="space-y-1">
          <Label htmlFor="preset-name">
            Name <span aria-hidden="true">*</span>
          </Label>
          <Input
            id="preset-name"
            aria-label="Name"
            maxLength={64}
            required
            value={attrs.name}
            onChange={(e) => onSetField("name", e.target.value)}
          />
        </div>
        <div className="space-y-1">
          <Label htmlFor="preset-default-name">Default character name</Label>
          <Input
            id="preset-default-name"
            aria-label="Default character name"
            value={attrs.defaultName}
            onChange={(e) => onSetField("defaultName", e.target.value)}
          />
          <p className="text-xs text-muted-foreground">
            empty = prompt on apply
          </p>
        </div>
        <div className="space-y-1 sm:col-span-2">
          <Label htmlFor="preset-description">Description</Label>
          <Input
            id="preset-description"
            aria-label="Description"
            maxLength={512}
            value={attrs.description}
            onChange={(e) => onSetField("description", e.target.value)}
          />
        </div>
        <div className="space-y-1 sm:col-span-2">
          <Label>Tags</Label>
          <div className="flex flex-row flex-wrap items-center gap-2">
            {attrs.tags.map((tag) => (
              <Button
                key={tag}
                type="button"
                variant="outline"
                size="sm"
                aria-label={`Remove tag ${tag}`}
                onClick={() => onRemoveTag(tag)}
              >
                {tag} <X className="ml-1 h-3 w-3" />
              </Button>
            ))}
            {addingTag ? (
              <div className="flex items-center gap-1">
                <Input
                  autoFocus
                  className="h-8 w-32"
                  placeholder="Tag..."
                  value={newTag}
                  onChange={(e) => setNewTag(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      e.preventDefault();
                      handleAddTag();
                    } else if (e.key === "Escape") {
                      setNewTag("");
                      setAddingTag(false);
                    }
                  }}
                />
                <Button type="button" size="sm" onClick={handleAddTag}>
                  Add
                </Button>
              </div>
            ) : (
              <Button
                type="button"
                variant="outline"
                size="icon"
                aria-label="Add tag"
                onClick={() => setAddingTag(true)}
              >
                <Plus className="h-4 w-4" />
              </Button>
            )}
          </div>
        </div>
      </div>
    </section>
  );
}
