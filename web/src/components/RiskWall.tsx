"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import type { RiskWallItem } from "@/lib/types";
import { getRiskWall } from "@/lib/api";

function scoreColor(score: number): string {
  if (score >= 8) return "text-risk-critical";
  if (score >= 6) return "text-risk-high";
  return "text-risk-medium";
}

function timeAgo(ts: number): string {
  const diff = Date.now() / 1000 - ts;
  if (diff < 3600) return `${Math.max(1, Math.floor(diff / 60))}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

export default function RiskWall() {
  const [items, setItems] = useState<RiskWallItem[] | null>(null);

  useEffect(() => {
    const load = () =>
      getRiskWall()
        .then(setItems)
        .catch(() => undefined);
    load();
    const timer = setInterval(load, 15_000);
    return () => clearInterval(timer);
  }, []);

  if (items === null) {
    // 骨架屏
    return (
      <section className="max-w-5xl mx-auto px-4 pb-16">
        <div className="h-7 w-48 bg-gray-800/60 rounded animate-pulse mb-6" />
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {[...Array(6)].map((_, i) => (
            <div
              key={i}
              className="gradient-card rounded-xl h-[76px] animate-pulse"
            />
          ))}
        </div>
      </section>
    );
  }

  if (items.length === 0) return null;

  return (
    <section className="max-w-5xl mx-auto px-4 pb-16">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-bold text-white flex items-center gap-2">
          🚨 High Risk Wall
          <span className="text-xs font-normal text-gray-500">
            auto-refresh 15s
          </span>
        </h2>
        <Link
          href="/history"
          className="text-sm text-gray-400 hover:text-purple-400 transition"
        >
          View all →
        </Link>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        {items.slice(0, 6).map((item) => (
          <Link
            key={item.token_address}
            href={`/scan/${item.token_address}`}
            className="gradient-card rounded-xl p-4 hover:border-red-500/40 transition group"
          >
            <div className="flex items-center justify-between mb-2">
              <div className="min-w-0">
                <span className="font-semibold text-white group-hover:text-red-400 transition">
                  {item.token_symbol}
                </span>
                <span className="text-xs text-gray-500 ml-2 truncate">
                  {item.token_name}
                </span>
              </div>
              <span className={`font-bold text-lg tabular-nums ${scoreColor(item.risk_score)}`}>
                {item.risk_score.toFixed(1)}
              </span>
            </div>
            <div className="flex items-center justify-between text-xs text-gray-500">
              <span className="truncate">{item.creator_label}</span>
              <span className="flex-shrink-0 ml-2">
                {item.scanned_at ? timeAgo(item.scanned_at) : ""}
              </span>
            </div>
          </Link>
        ))}
      </div>
    </section>
  );
}
