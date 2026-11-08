"use client";

import { useState, useCallback } from "react";
import type { ScanResponse } from "@/lib/types";
import { scanToken } from "@/lib/api";

interface UseScanReturn {
  result: ScanResponse | null;
  loading: boolean;
  error: string | null;
  scan: (tokenAddress: string) => Promise<void>;
  reset: () => void;
}

export function useScan(): UseScanReturn {
  const [result, setResult] = useState<ScanResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const scan = useCallback(async (tokenAddress: string) => {
    setLoading(true);
    setError(null);
    setResult(null);

    try {
      const data = await scanToken({
        token_address: tokenAddress,
        chain_id: 56,
      });
      setResult(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Scan failed");
    } finally {
      setLoading(false);
    }
  }, []);

  const reset = useCallback(() => {
    setResult(null);
    setError(null);
    setLoading(false);
  }, []);

  return { result, loading, error, scan, reset };
}
