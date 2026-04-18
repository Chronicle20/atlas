import { api } from "@/lib/api/client";
import { buildQueryString, type ServiceOptions, type QueryOptions } from "@/lib/api/query-params";
import type { ApiSingleResponse } from "@/types/api/responses";
import type {
  Service,
  ServiceType,
  ServiceAttributes,
  LoginService,
  ChannelService,
  DropsService,
  LoginServiceAttributes,
  ChannelServiceAttributes,
  DropsServiceAttributes,
  CreateServiceInput,
  UpdateServiceInput,
  TaskConfig,
  LoginTenant,
  ChannelTenant,
} from "@/types/models/service";

const BASE_PATH = "/api/configurations/services";

interface CreateServiceRequest {
  data: {
    id?: string;
    type: "services";
    attributes: ServiceAttributes;
  };
}

interface UpdateServiceRequest {
  data: {
    id: string;
    type: "services";
    attributes: ServiceAttributes;
  };
}

function transformServiceResponse(data: Record<string, unknown>): Service {
  const id = data.id as string;
  const attributes = data.attributes as Record<string, unknown> | undefined;

  const type = (attributes?.type ?? data.type) as ServiceType;
  const tasks = ((attributes?.tasks ?? data.tasks) as TaskConfig[]) || [];
  const tenants = attributes?.tenants ?? data.tenants;

  if (type === "login-service") {
    return {
      id,
      attributes: { type, tasks, tenants: (tenants as LoginTenant[]) || [] },
    } as LoginService;
  }
  if (type === "channel-service") {
    return {
      id,
      attributes: { type, tasks, tenants: (tenants as ChannelTenant[]) || [] },
    } as ChannelService;
  }
  return {
    id,
    attributes: { type: "drops-service", tasks },
  } as DropsService;
}

function buildAttributes(input: CreateServiceInput | UpdateServiceInput): ServiceAttributes {
  if (input.type === "login-service") {
    return {
      type: "login-service",
      tasks: input.tasks || [],
      tenants: (input.tenants as LoginTenant[]) || [],
    };
  }
  if (input.type === "channel-service") {
    return {
      type: "channel-service",
      tasks: input.tasks || [],
      tenants: (input.tenants as ChannelTenant[]) || [],
    };
  }
  return {
    type: "drops-service",
    tasks: input.tasks || [],
  };
}

export const servicesService = {
  async getAllServices(options?: QueryOptions): Promise<Service[]> {
    const response = await api.getList<Record<string, unknown>>(
      `${BASE_PATH}${buildQueryString(options)}`,
      options,
    );
    return response.map(transformServiceResponse);
  },

  async getServiceById(id: string, options?: ServiceOptions): Promise<Service> {
    const response = await api.getOne<Record<string, unknown>>(`${BASE_PATH}/${id}`, options);
    return transformServiceResponse(response);
  },

  async createService(input: CreateServiceInput, options?: ServiceOptions): Promise<Service> {
    const request: CreateServiceRequest = {
      data: { type: "services", attributes: buildAttributes(input) },
    };
    if (input.id) request.data.id = input.id;
    const response = await api.post<ApiSingleResponse<Record<string, unknown>>>(BASE_PATH, request, options);
    return transformServiceResponse(response.data);
  },

  async updateService(id: string, input: UpdateServiceInput, options?: ServiceOptions): Promise<Service> {
    const request: UpdateServiceRequest = {
      data: { id, type: "services", attributes: buildAttributes(input) },
    };
    const response = await api.patch<ApiSingleResponse<Record<string, unknown>>>(
      `${BASE_PATH}/${id}`,
      request,
      options,
    );
    return transformServiceResponse(response.data);
  },

  async deleteService(id: string, options?: ServiceOptions): Promise<void> {
    return api.delete(`${BASE_PATH}/${id}`, options);
  },
};

export type {
  Service,
  ServiceType,
  ServiceAttributes,
  LoginService,
  ChannelService,
  DropsService,
  LoginServiceAttributes,
  ChannelServiceAttributes,
  DropsServiceAttributes,
  CreateServiceInput,
  UpdateServiceInput,
  TaskConfig,
  LoginTenant,
  ChannelTenant,
};

export {
  isLoginService,
  isChannelService,
  isDropsService,
  getServiceTypeDisplayName,
  getServiceTenantCount,
  getServiceTaskCount,
  TASK_TYPES_BY_SERVICE,
} from "@/types/models/service";
