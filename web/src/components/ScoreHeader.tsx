"use client";

import { useEffect, useState } from "react";

interface ScoreHeaderProps {
  score: number;
  level: string;
  fromCache?: boolean;
}

function getTheme(score: number) {
  if (score <= 3)
    return { color: "#22c55e", text: "text-risk-low", emoji: "✅", ring: "border-green-500/30", bg: "from-green-500/15 to-transparent" };
  if (score <= 5)
    return { color: "#eab308", text: "text-risk-medium", emoji: "⚠️", ring: "border-yellow-500/30", bg: "from-yellow-500/15 to-transparent" };
  if (score <= 7)
    return { color: "#f97316", text: "text-risk-high", emoji: "🚨", ring: "border-orange-500/30", bg: "from-orange-500/15 to-transparent" };
  return { color: "#ef4444", text: "text-risk-critical", emoji: "💀", ring: "border-red-500/30", bg: "from-red-500/15 to-transparent" };
}

/** 分数滚动动画：0 → 最终分 */
function useScoreAnimation(target: number, duration = 1200): number {
  const [v, setV] = useState(0);
  useEffect(() => {
    const start = performance.now();
    let raf = 0;
    const tick = (now: number) => {
      const t = Math.min(1, (now - start) / duration);
      setV(target * (1 - Math.pow(1 - t, 3)));
      if (t < 1) raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [target, duration]);
  return v;
}

export default function ScoreHeader({ score, level, fromCache }: ScoreHeaderProps) {
  const theme = getTheme(score);
  const animated = useScoreAnimation(score);
  const CIRCUM = 2 * Math.PI * 54;

  return (
    <div
      className={`relative flex flex-col items-center py-8 mb-8 rounded-2xl
                  bg-gradient-to-b ${theme.bg} border ${theme.ring}
                  ${score >= 7 ? "glow-danger" : ""}`}
    >
      <div className="relative w-40 h-40">
        {/* SVG 评分环 */}
        <svg className="w-40 h-40 -rotate-90" viewBox="0 0 120 120">
          <circle
            cx="60" cy="60" r="54"
            fill="none" stroke="rgba(255,255,255,0.06)" strokeWidth="8"
          />
          <circle
            cx="60" cy="60" r="54"
            fill="none" stroke={theme.color} strokeWidth="8"
            strokeLinecap="round"
            strokeDasharray={`${(animated / 10) * CIRCUM} ${CIRCUM}`}
            style={{ filter: `drop-shadow(0 0 6px ${theme.color}66)` }}
          />
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className="text-3xl mb-1">{theme.emoji}</span>
          <span className={`text-5xl font-black tabular-nums ${theme.text}`}>
            {animated.toFixed(1)}
          </span>
          <span className="text-xs text-gray-500 mt-0.5">/ 10 risk</span>
        </div>
      </div>

      <div className={`text-2xl font-bold mt-3 ${theme.text}`}>{level}</div>
      <div className="text-xs text-gray-500 mt-1.5 flex items-center gap-2">
        <span>Higher score = more dangerous</span>
        {fromCache && (
          <span className="px-2 py-0.5 rounded-full bg-cyan-500/10 border border-cyan-500/30 text-cyan-300">
            ⚡ cached report
          </span>
        )}
      </div>
    </div>
  );
}
