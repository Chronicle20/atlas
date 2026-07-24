import { useParams } from "react-router-dom";
import { toast } from "sonner";
import { TemplateDetailLayout } from "@/components/features/templates/TemplateDetailLayout";
import {
  CharacterPresetsEditor,
  type PresetsEditorAdapter,
} from "@/components/features/characters/presets/CharacterPresetsEditor";
import { useTemplate, useUpdateTemplate } from "@/lib/hooks/api/useTemplates";

export function TemplatesCharacterPresetsPage() {
  const { id } = useParams();
  const templateQuery = useTemplate(String(id ?? ""));
  const updateTemplate = useUpdateTemplate();
  const template = templateQuery.data;

  const adapter: PresetsEditorAdapter = {
    presets: template?.attributes.characters.presets,
    isLoading: templateQuery.isLoading,
    error: templateQuery.error ?? null,
    isSaving: updateTemplate.isPending,
    save: (presets, onSuccess) => {
      if (!template) return;
      updateTemplate.mutate(
        {
          id: template.id,
          updates: {
            characters: { ...template.attributes.characters, presets },
          },
        },
        {
          onSuccess: (updated) => {
            toast.success("Successfully saved template.");
            onSuccess(updated.attributes.characters.presets);
          },
          onError: (error) =>
            toast.error(`Failed to update template: ${error.message}`),
        },
      );
    },
  };

  return (
    <TemplateDetailLayout>
      <CharacterPresetsEditor adapter={adapter} />
    </TemplateDetailLayout>
  );
}
