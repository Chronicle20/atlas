/**
 * Tenant Form Validation Schemas
 *
 * Zod schemas for validating tenant creation and related forms.
 */

import { z } from 'zod';

/**
 * Schema for creating a new tenant
 *
 * Validates:
 * - name: required, 1-100 characters
 * - region: required, 3 characters (e.g., "GMS", "JMS")
 * - majorVersion: required, non-negative integer
 * - minorVersion: required, non-negative integer
 */
export const createTenantSchema = z.object({
  name: z
    .string()
    .min(1, 'Tenant name is required')
    .max(100, 'Tenant name must be 100 characters or less'),
  region: z
    .string()
    .min(1, 'Region is required'),
  majorVersion: z
    .number()
    .int('Major version must be an integer')
    .nonnegative('Major version must be non-negative'),
  minorVersion: z
    .number()
    .int('Minor version must be an integer')
    .nonnegative('Minor version must be non-negative'),
});

/**
 * TypeScript type inferred from the create tenant schema
 */
export type CreateTenantFormData = z.infer<typeof createTenantSchema>;

/**
 * Default values for the create tenant form
 */
export const createTenantDefaults: CreateTenantFormData = {
  name: '',
  region: '',
  majorVersion: 0,
  minorVersion: 0,
};
