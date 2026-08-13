import { TemplateDetailLayout } from "@/components/features/templates/TemplateDetailLayout";
import { DefinitionGridPage } from "@/components/features/socket/DefinitionGridPage";

export function TemplatesHandlersPage() {
  return (
    <TemplateDetailLayout>
      <DefinitionGridPage kind="handler" scope="template" />
    </TemplateDetailLayout>
  );
}
