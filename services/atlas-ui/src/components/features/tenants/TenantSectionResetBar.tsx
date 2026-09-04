import {
  TenantResetButton,
  type TenantResetButtonProps,
} from "@/components/features/tenants/TenantResetButton";

/**
 * The single place scoped-reset placement is defined. Every sub-section that
 * offers a scoped reset renders this instead of hand-rolling its own wrapper,
 * so the button always sits top-right of the sub-section content with the
 * same spacing above the editor below it.
 */
export function TenantSectionResetBar(props: TenantResetButtonProps) {
  return (
    <div className="flex justify-end pb-4">
      <TenantResetButton {...props} />
    </div>
  );
}
