import { apiClient } from "./apiClient";
import type { ListProductsResult, Product, ProductPayload } from "./types";

export async function listProducts(): Promise<Product[]> {
  const res = await apiClient.get<ListProductsResult>("/products");
  return res.products;
}

export function fetchProduct(id: string): Promise<Product> {
  return apiClient.get<Product>(`/products/${id}`);
}

export function createProduct(payload: ProductPayload): Promise<Product> {
  return apiClient.post<Product>("/products", payload);
}

export function updateProduct(id: string, payload: ProductPayload): Promise<Product> {
  return apiClient.put<Product>(`/products/${id}`, payload);
}

export function deleteProduct(id: string): Promise<void> {
  return apiClient.del<void>(`/products/${id}`);
}
