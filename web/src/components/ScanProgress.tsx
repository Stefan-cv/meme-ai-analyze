"use client";

interface ScanProgressProps {
  progress: number;
  step: string;
}

/** 与后端 WS 推送的 8 个扫描阶段一一对应 */
const STEPS = [
  { icon: "📜", label: "Reading contract bytecode" },
  { icon: "🔒", label: "Checking LP lock status" },
  { icon: "👥", label: "Analyzing holder distribution" },
  { icon: "🕵️", label: "Tracking creator history" },
  { icon: "💧", label: "Evaluating liquidity depth" },
  { icon: "📣", label: "Fetching social signals" },
  { icon: "🤖", label: "AI generating report" },
  { icon: "⛓️", label: "Anchoring report on-chain" },
];

export default function ScanProgress({ progress, step }: ScanProgressProps) {
  // 根据 WS 推送的中文 step 文案匹配当前阶段索引
  const activeIndex = Math.min(
    STEPS.length - 1,
    Math.floor((progress / 100) * STEPS.length),
  );

  return (
    <div className="w-full max-w-lg mx-auto px-4 flex flex-col items-center gap-8 py-10">
      {/* 环形进度 */}
      <div className="relative w-28 h-28">
        <svg className="w-28 h-28" viewBox="0 0 100 100">
          <circle
            cx="50"
            cy="50"
            r="45"
            fill="none"
            stroke="rgba(100,100,200,0.1)"
            strokeWidth="6"
          />
          <circle
            cx="50"
            cy="50"
            r="45"
            fill="none"
            stroke="url(#scanGradient)"
            strokeWidth="6"
            strokeLinecap="round"
            strokeDasharray={`${(progress / 100) * 283} 283`}
            transform="rotate(-90 50 50)"
            className="transition-all duration-500 ease-out"
          />
          <defs>
            <linearGradient id="scanGradient" x1="0%" y1="0%" x2="100%" y2="0%">
              <stop offset="0%" stopColor="#a855f7" />
              <stop offset="100%" stopColor="#22d3ee" />
            </linearGradient>
          </defs>
        </svg>
        <div className="absolute inset-0 flex items-center justify-center">
          <span className="text-2xl font-bold text-white tabular-nums">
            {Math.round(progress)}%
          </span>
        </div>
      </div>

      {/* 步骤清单：完成打勾、当前高亮旋转 */}
      <ol className="w-full space-y-2.5">
        {STEPS.map((s, i) => {
          const done = i < activeIndex;
          const active = i === activeIndex;
          return (
            <li
              key={s.label}
              className={`flex items-center gap-3 px-4 py-2.5 rounded-xl border transition-all duration-300
                ${done
                  ? "border-green-500/20 bg-green-500/5 opacity-60"
                  : active
                    ? "border-purple-500/40 bg-purple-500/10"
                    : "border-gray-800/60 bg-gray-900/30 opacity-40"}`}
            >
              <span
                className={`w-5 h-5 flex items-center justify-center rounded-full text-xs flex-shrink-0
                  ${done
                    ? "bg-green-500/20 text-green-400"
                    : active
                      ? "bg-purple-500/20 text-purple-300"
                      : "bg-gray-800 text-gray-500"}`}
              >
                {done ? "✓" : i + 1}
              </span>
              <span className="text-sm">
                {s.icon} {s.label}
              </span>
              {active && (
                <span className="ml-auto inline-block w-3.5 h-3.5 border-2 border-purple-400/30 border-t-purple-300 rounded-full animate-spin flex-shrink-0" />
              )}
            </li>
          );
        })}
      </ol>

      <p className="text-xs text-gray-500 -mt-2">
        {step} · live progress via WebSocket
      </p>
    </div>
  );
}
