import { apiClient } from "@/services/apiClient";
import type { ApiSuccessEnvelope, PaginatedResult } from "@/types/api";

export type BillStatus = "UNPAID" | "PARTIALLY_PAID" | "PAID" | "CANCELLED";
export type BillItemCategory = "CONSULTATION" | "LAB" | "PHARMACY" | "PROCEDURE" | "ROOM_CHARGE" | "OTHER";
export type PaymentMethod = "CASH" | "CARD" | "UPI" | "BANK_TRANSFER" | "INSURANCE";

export interface BillItem {
  description: string;
  category: BillItemCategory;
  quantity: number;
  unitPrice: number;
  amount: number;
}

export interface Payment {
  id: string;
  billId: string;
  amount: number;
  method: PaymentMethod;
  referenceNo?: string;
  receivedByName?: string;
  paidAt: string;
}

export interface Bill {
  id: string;
  billNo: string;
  patientId: string;
  patientName: string;
  items: BillItem[];
  subtotal: number;
  discount: number;
  tax: number;
  totalAmount: number;
  amountPaid: number;
  balance: number;
  status: BillStatus;
  payments?: Payment[];
  createdAt: string;
  updatedAt: string;
}

export interface CreateBillItemPayload {
  description: string;
  category: BillItemCategory;
  quantity: number;
  unitPrice: number;
}

export interface CreateBillPayload {
  patientId: string;
  items: CreateBillItemPayload[];
  discount: number;
  tax: number;
}

export interface RecordPaymentPayload {
  amount: number;
  method: PaymentMethod;
  referenceNo?: string;
}

export interface SearchBillsParams {
  patientId?: string;
  status?: BillStatus;
  outstandingOnly?: boolean;
  page?: number;
  pageSize?: number;
}

async function search(params: SearchBillsParams): Promise<PaginatedResult<Bill>> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<PaginatedResult<Bill>>>("/billing/bills", { params });
  return data.data;
}

async function getById(id: string): Promise<Bill> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<Bill>>(`/billing/bills/${id}`);
  return data.data;
}

async function create(payload: CreateBillPayload): Promise<Bill> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<Bill>>("/billing/bills", payload);
  return data.data;
}

async function recordPayment(id: string, payload: RecordPaymentPayload): Promise<Bill> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<Bill>>(`/billing/bills/${id}/payments`, payload);
  return data.data;
}

async function cancel(id: string): Promise<Bill> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<Bill>>(`/billing/bills/${id}/cancel`);
  return data.data;
}

export const billingApi = { search, getById, create, recordPayment, cancel };
