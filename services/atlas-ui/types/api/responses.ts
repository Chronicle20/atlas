/**
 * Generic API response wrapper for all Atlas API endpoints.
 * 
 * The Atlas API follows JSON:API-like conventions with responses containing
 * a 'data' field for successful responses.
 */
export interface ApiResponse<T = unknown> {
  /** The response data - can be a single object or array depending on endpoint */
  data: T;
}

/**
 * API response for list endpoints that return multiple items
 */
export interface ApiListResponse<T = unknown> extends ApiResponse<T[]> {
  data: T[];
}

/**
 * API response for single item endpoints
 */
export interface ApiSingleResponse<T = unknown> extends ApiResponse<T> {
  data: T;
}

/**
 * API error response structure for failed requests
 */
export interface ApiErrorResponse {
  error: {
    /** Human-readable error message */
    detail: string;
    /** Error status code */
    status?: number;
    /** Error type/code for programmatic handling */
    code?: string;
  };
}

/**
 * Alternative error response format used by some endpoints
 */
export interface ApiSimpleErrorResponse {
  /** Simple error message */
  message: string;
}

/**
 * Union type for all possible API responses
 */
export type ApiResult<T = unknown> = ApiResponse<T> | ApiErrorResponse | ApiSimpleErrorResponse;

/**
 * Type guard to check if a response is an error response
 */
export function isApiErrorResponse(response: unknown): response is ApiErrorResponse {
  return typeof response === 'object' && 
         response !== null && 
         'error' in response &&
         typeof (response as ApiErrorResponse).error === 'object';
}

/**
 * Type guard to check if a response is a simple error response
 */
export function isApiSimpleErrorResponse(response: unknown): response is ApiSimpleErrorResponse {
  return typeof response === 'object' && 
         response !== null && 
         'message' in response &&
         typeof (response as ApiSimpleErrorResponse).message === 'string';
}

/**
 * Type guard to check if a response is a successful API response
 */
export function isApiSuccessResponse<T>(response: unknown): response is ApiResponse<T> {
  return typeof response === 'object' && 
         response !== null && 
         'data' in response &&
         !isApiErrorResponse(response) &&
         !isApiSimpleErrorResponse(response);
}

/**
 * Utility type for extracting the data type from an API response
 */
export type ExtractApiData<T> = T extends ApiResponse<infer U> ? U : never;