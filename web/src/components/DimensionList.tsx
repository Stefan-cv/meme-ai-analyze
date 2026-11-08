"use client";

import type { DimensionScore } from "@/lib/types";

interface DimensionListProps {
  dimensions: DimensionScore[];
}

function getScoreColor(score: number): string {
  if (score <= 3) return "text-green-400";
  if (score <= 6) return "text-yellow-400";
  return "text-red-400";
}

function getBarColor(score: number): string {
  if (score <= 3) return "bg-green-500";
  if (score <= 6) return "bg-yellow-500";
  return "bg-red-500";
}

function getStatusEmoji(status: string): string {
  switch (status) {
    case "good": return "✅";
    case "warning": return "⚠️";
    case "danger": return "🔴";
    default: return "❓";
  }
}

export default function DimensionList({ dimensions }: DimensionListProps) {
  return (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold text-gray-300 mb-4">
        Dimension Scores
      </h3>
      {dimensions.map((dim, i) => (
        <div key={i} className="gradient-card rounded-xl p-4">
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-2">
              <span>{getStatusEmoji(dim.status)}</span>
              <span className="font-semibold text-white">{dim.name}</span>
              <span className="text-xs text-gray-500">
                (weight: {(dim.weight * 100).toFixed(0)}%)
              </span>
            </div>
            <span className={`font-bold text-lg ${getScoreColor(dim.score)}`}>
              {dim.score}/10
            </span>
          </div>

          {/* Progress bar */}
          <div className="w-full h-2 bg-gray-800 rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full transition-all duration-500 ${getBarColor(dim.score)}`}
              style={{ width: `${(dim.score / 10) * 100}%` }}
            />
          </div>

          {/* Detail */}
          <div className="text-sm text-gray-400 mt-2">{dim.details}</div>
        </div>
      ))}
    </div>
  );
}
