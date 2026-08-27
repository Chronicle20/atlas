import { useParams } from "react-router-dom";
import { toast } from "sonner";
import { TemplateDetailLayout } from "@/components/features/templates/TemplateDetailLayout";
import {
  MapleLifeEditor,
  type MapleLifeEditorAdapter,
} from "@/components/features/characters/maple-life/MapleLifeEditor";
import { useTemplate, useUpdateTemplate } from "@/lib/hooks/api/useTemplates";

export function TemplatesMapleLifePage() {
  const { id } = useParams();
  const templateQuery = useTemplate(String(id ?? ""));
  const updateTemplate = useUpdateTemplate();
  const template = templateQuery.data;

  const adapter: MapleLifeEditorAdapter = {
    mapleLife: template?.attributes.mapleLife,
    isLoading: templateQuery.isLoading,
    error: templateQuery.error ?? null,
    isSaving: updateTemplate.isPending,
    save: (mapleLife, onSuccess) => {
      if (!template) return;
      updateTemplate.mutate(
        { id: template.id, updates: { ...template.attributes, mapleLife } },
        {
          onSuccess: () => {
            toast.success("Successfully saved Maple Life configuration.");
            onSuccess();
          },
          onError: (error) =>
            toast.error(
              `Failed to update Maple Life configuration: ${error.message}`,
            ),
        },
      );
    },
  };

  return (
    <TemplateDetailLayout>
      <MapleLifeEditor adapter={adapter} />
    </TemplateDetailLayout>
  );
}
