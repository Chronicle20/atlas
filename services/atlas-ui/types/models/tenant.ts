// Re-export the Tenant type from the lib/tenants module
// This centralizes all tenant-related types while maintaining the single source of truth
export type { Tenant, TenantBasic, TenantConfig, TenantBasicAttributes, TenantConfigAttributes } from "@/lib/tenants";