/**
 * Tenant Onboarding Service
 *
 * Coordinates the two-step tenant creation process:
 * 1. Create tenant in atlas-tenants (generates UUID)
 * 2. Create configuration in atlas-configurations with the tenant UUID + template data
 *
 * This service ensures both entry points (template grid and tenant switcher)
 * perform the complete tenant onboarding flow.
 */

import { tenantsService, type TenantBasic, type TenantConfig, type TenantConfigAttributes } from './tenants.service';
import { templatesService } from './templates.service';
import type { Template } from '@/types/models/template';

/**
 * Result of a successful tenant onboarding
 */
export interface OnboardResult {
  tenant: TenantBasic;
  config: TenantConfig;
}

/**
 * Error types for specific failure scenarios
 */
export class TemplateNotFoundError extends Error {
  constructor(region: string, majorVersion: number, minorVersion: number) {
    super(`Template not found for region=${region}, majorVersion=${majorVersion}, minorVersion=${minorVersion}`);
    this.name = 'TemplateNotFoundError';
  }
}

export class TenantCreationError extends Error {
  public readonly originalError?: unknown;

  constructor(message: string, originalError?: unknown) {
    super(message);
    this.name = 'TenantCreationError';
    this.originalError = originalError;
  }
}

export class ConfigurationCreationError extends Error {
  public readonly tenantId: string;
  public readonly originalError?: unknown;

  constructor(message: string, tenantId: string, originalError?: unknown) {
    super(message);
    this.name = 'ConfigurationCreationError';
    this.tenantId = tenantId;
    this.originalError = originalError;
  }
}

/**
 * Tenant Onboarding Service
 *
 * Provides a unified interface for tenant creation that ensures both
 * atlas-tenants and atlas-configurations entries are created correctly.
 */
class OnboardingService {
  /**
   * Onboard a new tenant using a pre-fetched template.
   *
   * Use this method when you already have the full template data
   * (e.g., from the template grid where templates are already loaded).
   *
   * @param name - The name for the new tenant
   * @param template - The full template to use for configuration
   * @returns OnboardResult with the created tenant and configuration
   * @throws TenantCreationError if tenant creation fails
   * @throws ConfigurationCreationError if config creation fails (includes tenant ID for recovery)
   */
  async onboardTenant(name: string, template: Template): Promise<OnboardResult> {
    // Step 1: Create tenant in atlas-tenants
    let tenant: TenantBasic;
    try {
      tenant = await tenantsService.createTenant({
        name,
        region: template.attributes.region,
        majorVersion: template.attributes.majorVersion,
        minorVersion: template.attributes.minorVersion,
      });
    } catch (error) {
      throw new TenantCreationError(
        'Failed to create tenant in atlas-tenants',
        error
      );
    }

    // Step 2: Create configuration in atlas-configurations with tenant ID
    let config: TenantConfig;
    try {
      // Build the configuration attributes from the template
      const configAttributes: TenantConfigAttributes = {
        region: template.attributes.region,
        majorVersion: template.attributes.majorVersion,
        minorVersion: template.attributes.minorVersion,
        usesPin: template.attributes.usesPin,
        characters: template.attributes.characters,
        npcs: template.attributes.npcs,
        socket: template.attributes.socket,
        worlds: template.attributes.worlds,
      };

      // Pass tenant ID to ensure configuration uses the same UUID
      config = await tenantsService.createTenantConfiguration(tenant.id, configAttributes);
    } catch (error) {
      // Provide tenant ID in error for potential manual recovery
      throw new ConfigurationCreationError(
        `Failed to create configuration in atlas-configurations. Tenant ${tenant.id} was created but has no configuration.`,
        tenant.id,
        error
      );
    }

    return { tenant, config };
  }

  /**
   * Onboard a new tenant by specifying template version.
   *
   * Use this method when you need to fetch the template by its version
   * (e.g., from the tenant switcher dialog with cascading dropdowns).
   *
   * @param name - The name for the new tenant
   * @param region - The template region (e.g., "GMS", "JMS")
   * @param majorVersion - The template major version
   * @param minorVersion - The template minor version
   * @returns OnboardResult with the created tenant and configuration
   * @throws TemplateNotFoundError if no template matches the specified version
   * @throws TenantCreationError if tenant creation fails
   * @throws ConfigurationCreationError if config creation fails
   */
  async onboardTenantByVersion(
    name: string,
    region: string,
    majorVersion: number,
    minorVersion: number
  ): Promise<OnboardResult> {
    // Fetch the template by version
    const templates = await templatesService.getByRegionAndVersion(
      region,
      majorVersion,
      minorVersion
    );

    const template = templates[0];
    if (!template) {
      throw new TemplateNotFoundError(region, majorVersion, minorVersion);
    }

    // Delegate to the main onboarding method
    return this.onboardTenant(name, template);
  }
}

// Export singleton instance
export const onboardingService = new OnboardingService();
