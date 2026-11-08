"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import type { RiskWallItem } from "@/lib/types";
import { getRiskWall } from "@/lib/api";

type SortKey = "score-desc" | "score-asc" | "recent";

function scoreColor(score: number): string {
  if (score >= 8) return "text-risk-critical";
  if (score >= 6) return "text-risk-high";
  if (score >= 4) return "text-risk-medium";
  return "text-risk-low";
}

function levelBadge(level: string): string {
  if (level.includes("极高") || level.includes("致命")) return "bg-red-500/15 text-red-300 border-red-500/30";
  if (level.includes("高")) return "bg-orange-500/15 text-orange-300 border-orange-500/30";
  if (level.includes("中")) return "bg-yellow-500/15 text-yellow-300 border-yellow-500/30";
  return "bg-green-500/15 text-green-300 border-green-500/30";
}

export default function HistoryPage() {
  const [items, setItems] = useState<RiskWallItem[] | null>(null);
  const [sort, setSort] = useState<SortKey>("score-desc");

  useEffect(() => {
    getRiskWall()
      .then(setItems)
      .catch(() => setItems([]));
  }, []);

  const sorted = useMemo(() => {
    if (!items) return [];
    return [...items].sort((a, b) => {
      if (sort === "score-asc") return a.risk_score - b.risk_score;
      if (sort === "recent") return b.scanned_at - a.scanned_at;
      return b.risk_score - a.risk_score;
    });
  }, [items, sort]);

  return (
    <div className="min-h-screen max-w-4xl mx-auto px-4 py-12">
      <div className="flex flex-wrap items-center gap-4 mb-8">
        <Link href="/" className="text-gray-400 hover:text-white transition">
          ← Back
        </Link>
        <h1 className="text-3xl font-bold text-white">Scan History</h1>
        <div className="flex gap-2 ml-auto">
          {(
            [
              ["score-desc", "Risk ↓"],
              ["score-asc", "Risk ↑"],
              ["recent", "Recent"],
            ] as [SortKey, string][]
          ).map(([k, label]) => (
            <button
              key={k}
              onClick={() => setSort(k)}
              className={`text-xs px-3 py-1.5 rounded-full border transition
                ${sort === k
                  ? "border-purple-500/50 bg-purple-500/15 text-purple-300"
                  : "border-gray-700 text-gray-400 hover:border-gray-500"}`}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {items === null ? (
        <div className="space-y-3">
          {[...Array(6)].map((_, i) => (
            <div key={i} className="gradient-card rounded-xl h-16 animate-pulse" />
          ))}
        </div>
      ) : sorted.length === 0 ? (
        <div className="text-center text-gray-400 py-12">
          No scan history yet.{" "}
          <Link href="/" className="text-purple-400 hover:underline">
            Scan a token →
          </Link>
        </div>
      ) : (
        <div className="space-y-3">
          {sorted.map((item) => (
            <Link
              key={item.token_address}
              href={`/scan/${item.token_address}`}
              className="block gradient-card rounded-xl p-4 hover:border-purple-500/40 transition group"
            >
              <div className="flex items-center justify-between gap-4">
                <div className="min-w-0">
                  <div className="font-semibold text-white group-hover:text-purple-300 transition">
                    {item.token_symbol}
                    <span className="text-xs text-gray-500 ml-2 font-normal">
                      {item.token_name}
                    </span>
                  </div>
                  <div className="text-xs text-gray-600 font-mono truncate mt-1">
                    {item.token_address}
                  </div>
                </div>
                <div className="flex items-center gap-3 flex-shrink-0">
                  <span
                    className={`text-xs px-2 py-0.5 rounded-full border ${levelBadge(item.risk_level)}`}
                  >
                    {item.risk_level}
                  </span>
                  <span className={`text-xl font-bold tabular-nums ${scoreColor(item.risk_score)}`}>
                    {item.risk_score.toFixed(1)}
                  </span>
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
