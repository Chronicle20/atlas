import { useParams } from "react-router-dom";
import { toast } from "sonner";
import { TemplateDetailLayout } from "@/components/features/templates/TemplateDetailLayout";
import {
  CharacterTemplatesEditor,
  type TemplatesEditorAdapter,
} from "@/components/features/characters/templates/CharacterTemplatesEditor";
import { useTemplate, useUpdateTemplate } from "@/lib/hooks/api/useTemplates";

export function TemplatesCharacterTemplatesPage() {
  const { id } = useParams();
  const templateQuery = useTemplate(String(id ?? ""));
  const updateTemplate = useUpdateTemplate();
  const template = templateQuery.data;

  const adapter: TemplatesEditorAdapter = {
    templates: template?.attributes.characters.templates,
    isLoading: templateQuery.isLoading,
    error: templateQuery.error ?? null,
    isSaving: updateTemplate.isPending,
    save: (templates, onSuccess) => {
      if (!template) return;
      updateTemplate.mutate(
        {
          id: template.id,
          updates: {
            characters: { ...template.attributes.characters, templates },
          },
        },
        {
          onSuccess: () => {
            toast.success("Successfully saved template.");
            onSuccess();
          },
          onError: (error) =>
            toast.error(`Failed to update template: ${error.message}`),
        },
      );
    },
  };

  return (
    <TemplateDetailLayout>
      <CharacterTemplatesEditor adapter={adapter} />
    </TemplateDetailLayout>
  );
}
