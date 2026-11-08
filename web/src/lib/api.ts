// ============================================
// API Client
// ============================================

import type {
  APIResponse,
  ScanRequest,
  ScanResponse,
  RiskWallItem,
  StatsResponse,
} from "./types";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function fetchAPI<T>(
  endpoint: string,
  options?: RequestInit
): Promise<APIResponse<T>> {
  const res = await fetch(`${API_URL}${endpoint}`, {
    headers: {
      "Content-Type": "application/json",
    },
    ...options,
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(
      (errorData as APIResponse).message || `HTTP ${res.status}`
    );
  }

  return res.json();
}

/** Submit a token address for risk scanning */
export async function scanToken(req: ScanRequest): Promise<ScanResponse> {
  const response = await fetchAPI<ScanResponse>("/api/v1/scan", {
    method: "POST",
    body: JSON.stringify(req),
  });
  if (response.code !== 0 || !response.data) {
    throw new Error(response.message || "Scan failed");
  }
  return response.data;
}

/** Get a cached report for a token */
export async function getReport(
  tokenAddress: string
): Promise<ScanResponse | null> {
  const response = await fetchAPI<ScanResponse>(
    `/api/v1/report/${tokenAddress}`
  );
  return response.data || null;
}

/** Get the risk wall (recent high-risk tokens) */
export async function getRiskWall(): Promise<RiskWallItem[]> {
  const response = await fetchAPI<RiskWallItem[]>("/api/v1/risk-wall");
  return response.data || [];
}

/** Get global platform statistics */
export async function getStats(): Promise<StatsResponse> {
  const response = await fetchAPI<StatsResponse>("/api/v1/stats");
  return (
    response.data || {
      total_scans: 0,
      total_reports: 0,
      high_risk_tokens: 0,
      rugged_creators: 0,
    }
  );
}
