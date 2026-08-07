import { TemplateDetailLayout } from "@/components/features/templates/TemplateDetailLayout";
import { DefinitionGridPage } from "@/components/features/socket/DefinitionGridPage";

export function TemplatesWritersPage() {
  return (
    <TemplateDetailLayout>
      <DefinitionGridPage kind="writer" scope="template" />
    </TemplateDetailLayout>
  );
}
