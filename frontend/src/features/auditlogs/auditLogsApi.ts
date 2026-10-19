import { apiClient } from "@/services/apiClient";
import type { ApiSuccessEnvelope, PaginatedResult } from "@/types/api";

export interface AuditLog {
  id: string;
  actorId?: string;
  actorEmail?: string;
  actorRole?: string;
  action: string;
  entityType: string;
  entityId?: string;
  metadata?: Record<string, unknown>;
  ip?: string;
  createdAt: string;
}

export interface SearchAuditLogsParams {
  actorId?: string;
  entityType?: string;
  entityId?: string;
  action?: string;
  from?: string;
  to?: string;
  page?: number;
  pageSize?: number;
}

async function search(params: SearchAuditLogsParams): Promise<PaginatedResult<AuditLog>> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<PaginatedResult<AuditLog>>>("/audit-logs", { params });
  return data.data;
}

export const auditLogsApi = { search };
