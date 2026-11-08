"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import type { ScanResponse } from "@/lib/types";
import { getReport, scanToken } from "@/lib/api";
import { useWebSocket } from "@/hooks/useWebSocket";
import ScoreHeader from "@/components/ScoreHeader";
import RadarChart from "@/components/RadarChart";
import DimensionList from "@/components/DimensionList";
import AIReportBlock from "@/components/AIReportBlock";
import CreatorTimeline from "@/components/CreatorTimeline";
import ScanProgress from "@/components/ScanProgress";

type Phase = "checking-cache" | "scanning" | "done" | "error";

export default function ReportPage() {
  const params = useParams();
  const tokenAddress = params.tokenAddress as string;

  const [phase, setPhase] = useState<Phase>("checking-cache");
  const [result, setResult] = useState<ScanResponse | null>(null);
  const [fromCache, setFromCache] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [restFallback, setRestFallback] = useState(false);
  // REST 降级模式下的估算进度
  const [restProgress, setRestProgress] = useState(0);
  const fallbackTimer = useRef<ReturnType<typeof setInterval> | null>(null);

  const { progress: wsProgress, step: wsStep, result: wsResult, error: wsError, startScan } =
    useWebSocket();

  /** REST 降级扫描（WS 不可用时） */
  const restScan = useCallback(async (addr: string) => {
    setRestFallback(true);
    setPhase("scanning");
    // 扫描通常 15-30s，用 90s 渐近进度估算
    let p = 0;
    fallbackTimer.current = setInterval(() => {
      p = Math.min(92, p + Math.random() * 6);
      setRestProgress(p);
    }, 1200);
    try {
      const data = await scanToken({ token_address: addr, chain_id: 56 });
      if (fallbackTimer.current) clearInterval(fallbackTimer.current);
      setResult(data);
      setPhase("done");
    } catch (e) {
      if (fallbackTimer.current) clearInterval(fallbackTimer.current);
      setError(e instanceof Error ? e.message : "Scan failed");
      setPhase("error");
    }
  }, []);

  useEffect(() => {
    if (!tokenAddress) return;
    let cancelled = false;

    (async () => {
      setPhase("checking-cache");
      // ① 先查缓存报告：命中直接展示（后端 Redis + PostgreSQL 风险感知缓存）
      try {
        const cached = await getReport(tokenAddress);
        if (cancelled) return;
        if (cached) {
          setResult(cached);
          setFromCache(true);
          setPhase("done");
          return;
        }
      } catch {
        /* 缓存查询失败继续走扫描 */
      }

      // ② WebSocket 实时扫描（真实阶段进度）
      if (typeof WebSocket !== "undefined") {
        setPhase("scanning");
        startScan(tokenAddress);
      } else {
        await restScan(tokenAddress);
      }
    })();

    return () => {
      cancelled = true;
      if (fallbackTimer.current) clearInterval(fallbackTimer.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tokenAddress]);

  // WS 结果 / 失败降级
  useEffect(() => {
    if (wsResult) {
      setResult(wsResult);
      setFromCache(false);
      setPhase("done");
    }
  }, [wsResult]);

  useEffect(() => {
    if (wsError && phase === "scanning" && !restFallback && !result) {
      // ③ WS 失败 → 自动降级 REST，保证可用性
      restScan(tokenAddress);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [wsError]);

  if (phase === "checking-cache") {
    return (
      <div className="min-h-[60vh] flex flex-col items-center justify-center gap-4">
        <span className="w-10 h-10 border-2 border-purple-500/30 border-t-purple-400 rounded-full animate-spin" />
        <p className="text-gray-400 text-sm">Checking cached reports…</p>
      </div>
    );
  }

  if (phase === "scanning") {
    const progress = restFallback ? restProgress : wsProgress;
    const step = restFallback ? "REST fallback scanning…" : wsStep;
    return (
      <div className="min-h-[70vh] flex items-center justify-center">
        <ScanProgress progress={progress} step={step} />
      </div>
    );
  }

  if (phase === "error") {
    return (
      <div className="min-h-[60vh] flex flex-col items-center justify-center gap-4">
        <div className="text-5xl">⚠️</div>
        <div className="text-red-400 text-lg">{error}</div>
        <div className="flex gap-3">
          <button
            onClick={() => window.location.reload()}
            className="px-6 py-2.5 bg-purple-600 hover:bg-purple-500 rounded-lg transition text-sm font-medium"
          >
            Retry
          </button>
          <Link
            href="/"
            className="px-6 py-2.5 border border-gray-700 hover:border-gray-500 rounded-lg transition text-sm"
          >
            Back to scanner
          </Link>
        </div>
      </div>
    );
  }

  if (!result) return null;

  return (
    <div className="min-h-screen max-w-5xl mx-auto px-4 py-8 animate-fade-in">
      {/* Token Header */}
      <div className="flex flex-col items-center gap-3 mb-6">
        <div className="flex items-center gap-3 flex-wrap justify-center">
          <h2 className="text-2xl font-bold text-white">
            {result.token_name}{" "}
            <span className="text-gray-400">(${result.token_symbol})</span>
          </h2>
          <button
            onClick={() => navigator.clipboard.writeText(result.token_address)}
            title="Copy address"
            className="text-xs font-mono px-2.5 py-1 rounded-lg border border-gray-700
                       text-gray-400 hover:text-purple-300 hover:border-purple-500/40 transition"
          >
            {result.token_address.slice(0, 8)}…{result.token_address.slice(-6)} 📋
          </button>
        </div>
        <ScoreHeader score={result.risk_score} level={result.risk_level} fromCache={fromCache} />
      </div>

      {/* Radar + Dimensions */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-8 mb-8">
        <RadarChart dimensions={result.dimensions} />
        <DimensionList dimensions={result.dimensions} />
      </div>

      {/* AI Report */}
      <AIReportBlock
        report={result.ai_report}
        reportHash={result.report_hash}
        txHash={result.tx_hash}
      />

      {/* Creator History */}
      {result.creator_history && (
        <CreatorTimeline creator={result.creator_history} />
      )}

      {/* Action Buttons */}
      <div className="flex flex-wrap gap-4 justify-center mt-8 pb-8">
        {result.tx_hash && (
          <a
            href={`https://bscscan.com/tx/${result.tx_hash}`}
            target="_blank"
            rel="noopener noreferrer"
            className="px-6 py-3 bg-blue-600/20 border border-blue-500/30 hover:bg-blue-600/30 rounded-lg transition text-sm"
          >
            🔗 View on-chain proof
          </a>
        )}
        <button
          onClick={() => window.location.reload()}
          className="px-6 py-3 bg-purple-600/20 border border-purple-500/30 hover:bg-purple-600/30 rounded-lg transition text-sm"
        >
          🔄 Re-analyze
        </button>
        <Link
          href="/"
          className="px-6 py-3 border border-gray-700 hover:border-gray-500 rounded-lg transition text-sm"
        >
          ← Scan another token
        </Link>
      </div>

      {/* Disclaimer */}
      <div className="text-center text-xs text-gray-600 pb-8">
        This analysis is generated by AI. Meme tokens are inherently high-risk.
        Always DYOR. Not financial advice.
      </div>
    </div>
  );
}
