import { apiClient } from "@/services/apiClient";
import type { ApiSuccessEnvelope, PaginatedResult } from "@/types/api";

export type MedicineStatus = "ACTIVE" | "INACTIVE";
export type MovementType = "IN" | "OUT";

export interface Medicine {
  id: string;
  name: string;
  genericName?: string;
  category?: string;
  manufacturer?: string;
  unit?: string;
  reorderLevel: number;
  currentStock: number;
  isLowStock: boolean;
  status: MedicineStatus;
  createdAt: string;
  updatedAt: string;
}

export interface CreateMedicinePayload {
  name: string;
  genericName?: string;
  category?: string;
  manufacturer?: string;
  unit?: string;
  reorderLevel: number;
}

export interface UpdateMedicinePayload {
  name?: string;
  genericName?: string;
  category?: string;
  manufacturer?: string;
  unit?: string;
  reorderLevel?: number;
  status?: MedicineStatus;
}

export interface SearchMedicinesParams {
  query?: string;
  lowStock?: boolean;
  page?: number;
  pageSize?: number;
}

export interface Batch {
  id: string;
  medicineId: string;
  medicineName: string;
  batchNo: string;
  expiryDate: string;
  quantity: number;
  isExpired: boolean;
  purchasePrice: number;
  sellingPrice: number;
  createdAt: string;
}

export interface ReceiveBatchPayload {
  batchNo: string;
  expiryDate: string;
  quantity: number;
  purchasePrice: number;
  sellingPrice: number;
}

export interface BatchAllocation {
  batchId: string;
  batchNo: string;
  quantity: number;
}

export interface DispensedItem {
  medicineId: string;
  medicineName: string;
  quantity: number;
  batches: BatchAllocation[];
}

export interface DispenseItemPayload {
  medicineId: string;
  quantity: number;
}

export interface DispensePayload {
  prescriptionId: string;
  items: DispenseItemPayload[];
}

export interface DispenseResult {
  prescriptionId: string;
  dispensedAt: string;
  items: DispensedItem[];
}

export interface StockMovement {
  id: string;
  medicineId: string;
  medicineName: string;
  batchId: string;
  batchNo: string;
  type: MovementType;
  quantity: number;
  reason?: string;
  prescriptionId?: string;
  performedByName?: string;
  createdAt: string;
}

export interface SearchStockMovementsParams {
  medicineId?: string;
  type?: MovementType;
  page?: number;
  pageSize?: number;
}

async function searchMedicines(params: SearchMedicinesParams): Promise<PaginatedResult<Medicine>> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<PaginatedResult<Medicine>>>("/pharmacy/medicines", { params });
  return data.data;
}

async function getMedicineById(id: string): Promise<Medicine> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<Medicine>>(`/pharmacy/medicines/${id}`);
  return data.data;
}

async function createMedicine(payload: CreateMedicinePayload): Promise<Medicine> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<Medicine>>("/pharmacy/medicines", payload);
  return data.data;
}

async function updateMedicine(id: string, payload: UpdateMedicinePayload): Promise<Medicine> {
  const { data } = await apiClient.patch<ApiSuccessEnvelope<Medicine>>(`/pharmacy/medicines/${id}`, payload);
  return data.data;
}

async function listBatches(medicineId: string): Promise<Batch[]> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<{ batches: Batch[] }>>(`/pharmacy/medicines/${medicineId}/batches`);
  return data.data.batches;
}

async function receiveBatch(medicineId: string, payload: ReceiveBatchPayload): Promise<Batch> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<Batch>>(`/pharmacy/medicines/${medicineId}/batches`, payload);
  return data.data;
}

async function listExpiringBatches(withinDays?: number): Promise<Batch[]> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<{ batches: Batch[] }>>("/pharmacy/batches/expiring", {
    params: withinDays ? { withinDays } : undefined,
  });
  return data.data.batches;
}

async function dispense(payload: DispensePayload): Promise<DispenseResult> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<DispenseResult>>("/pharmacy/dispense", payload);
  return data.data;
}

async function searchStockMovements(params: SearchStockMovementsParams): Promise<PaginatedResult<StockMovement>> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<PaginatedResult<StockMovement>>>("/pharmacy/stock-movements", { params });
  return data.data;
}

export const pharmacyApi = {
  searchMedicines,
  getMedicineById,
  createMedicine,
  updateMedicine,
  listBatches,
  receiveBatch,
  listExpiringBatches,
  dispense,
  searchStockMovements,
};
