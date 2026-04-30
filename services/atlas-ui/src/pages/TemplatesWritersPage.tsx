import { TemplateDetailLayout } from "@/components/features/templates/TemplateDetailLayout";
import { WritersForm } from "@/pages/templates-writers-form";

export function TemplatesWritersPage() {
    return (
        <TemplateDetailLayout>
            <WritersForm />
        </TemplateDetailLayout>
    );
}
