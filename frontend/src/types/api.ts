/**
 * Mirrors the backend's response envelope (backend/internal/response) so
 * every API call in the app is typed consistently.
 */

export interface ApiMeta {
  requestId?: string;
}

export interface ApiErrorBody {
  code: string;
  message: string;
  details?: unknown;
}

export interface ApiSuccessEnvelope<T> {
  success: true;
  data: T;
  meta?: ApiMeta;
}

export interface ApiErrorEnvelope {
  success: false;
  error: ApiErrorBody;
  meta?: ApiMeta;
}

export type ApiEnvelope<T> = ApiSuccessEnvelope<T> | ApiErrorEnvelope;

export interface PaginatedResult<T> {
  items: T[];
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
}
