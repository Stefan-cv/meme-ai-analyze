"use client";

import { useEffect, useRef, useState } from "react";
import type { StatsResponse } from "@/lib/types";
import { getStats } from "@/lib/api";

/** 数字滚动动画（面试细节：数值变化时平滑过渡） */
function useCountUp(target: number, duration = 800): number {
  const [value, setValue] = useState(0);
  const fromRef = useRef(0);

  useEffect(() => {
    const from = fromRef.current;
    if (from === target) return;
    const start = performance.now();
    let raf = 0;
    const tick = (now: number) => {
      const t = Math.min(1, (now - start) / duration);
      const eased = 1 - Math.pow(1 - t, 3);
      setValue(Math.round(from + (target - from) * eased));
      if (t < 1) raf = requestAnimationFrame(tick);
      else fromRef.current = target;
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [target, duration]);

  return value;
}

function StatCard({
  icon,
  label,
  value,
}: {
  icon: string;
  label: string;
  value: number;
}) {
  const display = useCountUp(value);
  return (
    <div className="gradient-card rounded-xl p-4 text-center hover:border-purple-500/30 transition">
      <div className="text-2xl mb-1">{icon}</div>
      <div className="text-2xl font-bold text-white tabular-nums">
        {display.toLocaleString()}
      </div>
      <div className="text-xs text-gray-500 mt-1">{label}</div>
    </div>
  );
}

export default function StatsBar() {
  const [stats, setStats] = useState<StatsResponse>({
    total_scans: 0,
    total_reports: 0,
    high_risk_tokens: 0,
    rugged_creators: 0,
  });

  // 10s 轮询保持平台数据新鲜
  useEffect(() => {
    const load = () => getStats().then(setStats).catch(() => undefined);
    load();
    const timer = setInterval(load, 10_000);
    return () => clearInterval(timer);
  }, []);

  return (
    <div className="max-w-5xl mx-auto px-4 pb-12">
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard icon="🔍" label="Tokens Analyzed" value={stats.total_scans} />
        <StatCard icon="📊" label="Reports Generated" value={stats.total_reports} />
        <StatCard icon="🚨" label="High Risk Found" value={stats.high_risk_tokens} />
        <StatCard icon="💀" label="Rug Creators" value={stats.rugged_creators} />
      </div>
    </div>
  )
}
