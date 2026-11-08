// ============================================
// Frontend API Types (mirrors backend types)
// ============================================

export interface ScanRequest {
  token_address: string;
  chain_id?: number;
}

export interface ScanResponse {
  token_address: string;
  token_name: string;
  token_symbol: string;
  risk_score: number;
  risk_level: string;
  dimensions: DimensionScore[];
  ai_report: string;
  creator_history: CreatorProfile | null;
  report_hash: string;
  tx_hash: string;
  scanned_at: number;
}

export interface DimensionScore {
  name: string;
  score: number;
  weight: number;
  details: string;
  status: "good" | "warning" | "danger";
}

export interface CreatorProfile {
  address: string;
  total_tokens: number;
  active_tokens: number;
  rugged_tokens: number;
  rug_rate: number;
  avg_lifetime_hrs: number;
  first_seen_days: number;
  risk_label: string;
  recent_tokens: TokenBrief[];
}

export interface TokenBrief {
  address: string;
  name: string;
  symbol: string;
  status: "active" | "rugged" | "unknown";
  lifetime_hrs: number;
  created_at: number;
}

export interface WSMessage {
  status: "scanning" | "completed" | "error";
  step?: string;
  progress?: number;
  data?: ScanResponse;
  error?: string;
}

export interface StatsResponse {
  total_scans: number;
  total_reports: number;
  high_risk_tokens: number;
  rugged_creators: number;
}

export interface RiskWallItem {
  token_address: string;
  token_name: string;
  token_symbol: string;
  risk_score: number;
  risk_level: string;
  creator_label: string;
  scanned_at: number;
}

export interface APIResponse<T = unknown> {
  code: number;
  message: string;
  data?: T;
}
