"use client";

import type { CreatorProfile } from "@/lib/types";

interface CreatorTimelineProps {
  creator: CreatorProfile;
}

function getStatusIcon(status: string): string {
  switch (status) {
    case "active": return "🟢";
    case "rugged": return "💀";
    default: return "❓";
  }
}

export default function CreatorTimeline({ creator }: CreatorTimelineProps) {
  return (
    <div className="gradient-card rounded-xl p-6 mb-8">
      <div className="flex items-center gap-2 mb-4">
        <span className="text-xl">📋</span>
        <h3 className="text-lg font-semibold text-white">Creator History</h3>
      </div>

      {/* Creator summary */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900/50 rounded-lg p-3 text-center">
          <div className="text-2xl font-bold text-white">
            {creator.total_tokens}
          </div>
          <div className="text-xs text-gray-500">Total Tokens</div>
        </div>
        <div className="bg-gray-900/50 rounded-lg p-3 text-center">
          <div className="text-2xl font-bold text-red-400">
            {creator.rugged_tokens}
          </div>
          <div className="text-xs text-gray-500">Rugged</div>
        </div>
        <div className="bg-gray-900/50 rounded-lg p-3 text-center">
          <div className="text-2xl font-bold text-orange-400">
            {(creator.rug_rate * 100).toFixed(0)}%
          </div>
          <div className="text-xs text-gray-500">Rug Rate</div>
        </div>
        <div className="bg-gray-900/50 rounded-lg p-3 text-center">
          <div className="text-2xl font-bold text-cyan-400">
            {creator.avg_lifetime_hrs.toFixed(1)}h
          </div>
          <div className="text-xs text-gray-500">Avg Lifetime</div>
        </div>
      </div>

      {/* Risk label */}
      <div className="bg-gray-900/50 rounded-lg p-4 mb-4 text-center">
        <span className="text-sm">{creator.risk_label}</span>
      </div>

      {/* Creator address */}
      <div className="mb-4">
        <span className="text-xs text-gray-500">Creator Address: </span>
        <span className="text-sm font-mono text-gray-300 break-all">
          {creator.address}
        </span>
      </div>

      {/* Recent tokens */}
      {creator.recent_tokens.length > 0 && (
        <>
          <h4 className="text-sm font-semibold text-gray-400 mb-3">
            Recent Tokens ({creator.recent_tokens.length})
          </h4>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-gray-500 border-b border-gray-700/50">
                  <th className="text-left py-2 px-2">Token</th>
                  <th className="text-center py-2 px-2">Status</th>
                  <th className="text-right py-2 px-2">Lifetime</th>
                  <th className="text-right py-2 px-2">Created</th>
                </tr>
              </thead>
              <tbody>
                {creator.recent_tokens.map((token) => (
                  <tr
                    key={token.address}
                    className="border-b border-gray-800/50 hover:bg-gray-800/30"
                  >
                    <td className="py-2 px-2">
                      <div className="font-medium text-gray-200">
                        {token.name || token.symbol || "Unknown"}
                      </div>
                      <div className="text-xs text-gray-600 font-mono">
                        {token.address.slice(0, 8)}...
                      </div>
                    </td>
                    <td className="text-center py-2 px-2">
                      <span title={token.status}>
                        {getStatusIcon(token.status)}
                      </span>
                    </td>
                    <td className="text-right py-2 px-2 text-gray-400">
                      {token.lifetime_hrs.toFixed(1)}h
                    </td>
                    <td className="text-right py-2 px-2 text-gray-500 text-xs">
                      {new Date(token.created_at * 1000).toLocaleDateString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  );
}
