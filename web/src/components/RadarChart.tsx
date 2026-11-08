"use client";

import type { DimensionScore } from "@/lib/types";

interface RadarChartProps {
  dimensions: DimensionScore[];
}

const DIM_LABELS = [
  "Contract",
  "LP Lock",
  "Holders",
  "Creator",
  "Liquidity",
  "Social",
];

const CENTER = 150;
const RADIUS = 120;
const LEVELS = 5;

function polarToCartesian(
  cx: number,
  cy: number,
  r: number,
  angleDeg: number
): { x: number; y: number } {
  const rad = ((angleDeg - 90) * Math.PI) / 180;
  return {
    x: cx + r * Math.cos(rad),
    y: cy + r * Math.sin(rad),
  };
}

function getColor(score: number): string {
  if (score <= 3) return "#22c55e";
  if (score <= 6) return "#eab308";
  if (score <= 8) return "#f97316";
  return "#ef4444";
}

export default function RadarChart({ dimensions }: RadarChartProps) {
  const numDims = dimensions.length;
  const angleStep = 360 / numDims;

  // Build the data polygon points
  const dataPoints = dimensions.map((d, i) => {
    const r = (d.score / 10) * RADIUS;
    return polarToCartesian(CENTER, CENTER, r, i * angleStep);
  });
  const dataPath = dataPoints
    .map((p, i) => `${i === 0 ? "M" : "L"} ${p.x} ${p.y}`)
    .join(" ") + " Z";

  // Build level grid circles
  const gridCircles = Array.from({ length: LEVELS }, (_, i) => {
    const r = (RADIUS / LEVELS) * (i + 1);
    return r;
  });

  // Build axis lines
  const axisLines = dimensions.map((_, i) => {
    const end = polarToCartesian(CENTER, CENTER, RADIUS, i * angleStep);
    return { x: end.x, y: end.y };
  });

  return (
    <div className="flex flex-col items-center">
      <svg
        viewBox={`0 0 ${CENTER * 2} ${CENTER * 2}`}
        className="w-full max-w-[300px] h-auto"
      >
        {/* Grid circles */}
        {gridCircles.map((r, i) => (
          <circle
            key={i}
            cx={CENTER}
            cy={CENTER}
            r={r}
            fill="none"
            stroke="rgba(100,100,200,0.15)"
            strokeWidth="1"
          />
        ))}

        {/* Axis lines */}
        {axisLines.map((end, i) => (
          <line
            key={i}
            x1={CENTER}
            y1={CENTER}
            x2={end.x}
            y2={end.y}
            stroke="rgba(100,100,200,0.2)"
            strokeWidth="1"
          />
        ))}

        {/* Data polygon */}
        <path
          d={dataPath}
          fill="rgba(139,92,246,0.2)"
          stroke="rgba(139,92,246,0.6)"
          strokeWidth="2"
        />

        {/* Data points */}
        {dataPoints.map((p, i) => (
          <circle
            key={i}
            cx={p.x}
            cy={p.y}
            r="4"
            fill={getColor(dimensions[i].score)}
            stroke="white"
            strokeWidth="1"
          />
        ))}

        {/* Labels */}
        {dimensions.map((d, i) => {
          const labelPos = polarToCartesian(
            CENTER,
            CENTER,
            RADIUS + 22,
            i * angleStep
          );
          return (
            <text
              key={i}
              x={labelPos.x}
              y={labelPos.y}
              textAnchor="middle"
              dominantBaseline="middle"
              fill={getColor(d.score)}
              fontSize="11"
              fontWeight="600"
            >
              {DIM_LABELS[i]}
            </text>
          );
        })}
      </svg>
    </div>
  );
}
