"use client";

import { useState, useRef, useCallback, useEffect } from "react";
import type { WSMessage, ScanResponse } from "@/lib/types";

const WS_URL = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080";

interface UseWebSocketReturn {
  progress: number;
  step: string;
  result: ScanResponse | null;
  error: string | null;
  isScanning: boolean;
  startScan: (tokenAddress: string) => void;
}

export function useWebSocket(): UseWebSocketReturn {
  const [progress, setProgress] = useState(0);
  const [step, setStep] = useState("");
  const [result, setResult] = useState<ScanResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isScanning, setIsScanning] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);

  const startScan = useCallback((tokenAddress: string) => {
    // Reset state
    setProgress(0);
    setStep("正在连接...");
    setResult(null);
    setError(null);
    setIsScanning(true);

    // Close existing connection
    if (wsRef.current) {
      wsRef.current.close();
    }

    const ws = new WebSocket(`${WS_URL}/ws/scan?token=${tokenAddress}`);
    wsRef.current = ws;

    ws.onmessage = (event) => {
      const msg: WSMessage = JSON.parse(event.data);

      switch (msg.status) {
        case "scanning":
          setStep(msg.step || "");
          setProgress(msg.progress || 0);
          break;
        case "completed":
          setIsScanning(false);
          if (msg.data) {
            setResult(msg.data);
            setProgress(100);
            setStep("分析完成");
          }
          ws.close();
          break;
        case "error":
          setIsScanning(false);
          setError(msg.error || "Unknown error");
          ws.close();
          break;
      }
    };

    ws.onerror = () => {
      setIsScanning(false);
      setError("WebSocket connection failed");
    };

    ws.onclose = () => {
      setIsScanning(false);
    };
  }, []);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, []);

  return { progress, step, result, error, isScanning, startScan };
}
