/**
 * Service Configuration Validation Schemas
 *
 * Zod schemas for validating service creation and configuration.
 */

import { z } from 'zod';

/**
 * Valid service types
 */
export const serviceTypeSchema = z.enum([
  'login-service',
  'channel-service',
  'drops-service',
]);

/**
 * Task configuration schema
 */
export const taskConfigSchema = z.object({
  type: z.string().min(1, 'Task type is required'),
  interval: z
    .number()
    .int('Interval must be an integer')
    .nonnegative('Interval must be non-negative'),
  duration: z
    .number()
    .int('Duration must be an integer')
    .nonnegative('Duration must be non-negative'),
});

/**
 * Port validation (1-65535)
 */
export const portSchema = z
  .number()
  .int('Port must be an integer')
  .min(1, 'Port must be at least 1')
  .max(65535, 'Port must be at most 65535');

/**
 * Byte ID validation (0-255) for world/channel IDs
 */
export const byteIdSchema = z
  .number()
  .int('ID must be an integer')
  .min(0, 'ID must be at least 0')
  .max(255, 'ID must be at most 255');

/**
 * Login tenant configuration schema
 */
export const loginTenantSchema = z.object({
  id: z.string().uuid('Invalid tenant ID format'),
  port: portSchema,
});

/**
 * Channel configuration schema
 */
export const channelChannelSchema = z.object({
  id: byteIdSchema,
  port: portSchema,
});

/**
 * World configuration schema
 */
export const channelWorldSchema = z.object({
  id: byteIdSchema,
  channels: z.array(channelChannelSchema),
});

/**
 * Channel tenant configuration schema
 */
export const channelTenantSchema = z.object({
  id: z.string().uuid('Invalid tenant ID format'),
  ipAddress: z.string().min(1, 'IP address is required'),
  worlds: z.array(channelWorldSchema),
});

/**
 * Login service schema
 */
export const loginServiceSchema = z.object({
  id: z.string().uuid().optional(),
  type: z.literal('login-service'),
  tasks: z.array(taskConfigSchema),
  tenants: z.array(loginTenantSchema),
});

/**
 * Channel service schema
 */
export const channelServiceSchema = z.object({
  id: z.string().uuid().optional(),
  type: z.literal('channel-service'),
  tasks: z.array(taskConfigSchema),
  tenants: z.array(channelTenantSchema),
});

/**
 * Drops service schema
 */
export const dropsServiceSchema = z.object({
  id: z.string().uuid().optional(),
  type: z.literal('drops-service'),
  tasks: z.array(taskConfigSchema),
});

/**
 * Combined service schema using discriminated union
 */
export const serviceSchema = z.discriminatedUnion('type', [
  loginServiceSchema,
  channelServiceSchema,
  dropsServiceSchema,
]);

/**
 * Create service input schema
 */
export const createServiceInputSchema = z.object({
  id: z.string().uuid('Invalid service ID format').optional(),
  type: serviceTypeSchema,
  tasks: z.array(taskConfigSchema),
  tenants: z
    .union([z.array(loginTenantSchema), z.array(channelTenantSchema)])
    .optional(),
});

/**
 * Validation helper for unique world IDs within a tenant
 */
export function validateUniqueWorldIds(worlds: { id: number }[]): boolean {
  const ids = worlds.map((w) => w.id);
  return new Set(ids).size === ids.length;
}

/**
 * Validation helper for unique channel IDs within a world
 */
export function validateUniqueChannelIds(channels: { id: number }[]): boolean {
  const ids = channels.map((c) => c.id);
  return new Set(ids).size === ids.length;
}

/**
 * TypeScript types inferred from schemas
 */
export type ServiceType = z.infer<typeof serviceTypeSchema>;
export type TaskConfig = z.infer<typeof taskConfigSchema>;
export type LoginTenant = z.infer<typeof loginTenantSchema>;
export type ChannelChannel = z.infer<typeof channelChannelSchema>;
export type ChannelWorld = z.infer<typeof channelWorldSchema>;
export type ChannelTenant = z.infer<typeof channelTenantSchema>;
export type CreateServiceInput = z.infer<typeof createServiceInputSchema>;
