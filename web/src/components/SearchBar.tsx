"use client";

import { useState } from "react";

interface SearchBarProps {
  onSearch: (address: string) => void;
  loading: boolean;
}

/** 预置示例：知名 BSC 代币（低风险对照） */
const EXAMPLES = [
  { symbol: "CAKE", address: "0x0e09fabb73bd3ade0a17ecc321fd13a19e81ce82" },
];

export default function SearchBar({ onSearch, loading }: SearchBarProps) {
  const [value, setValue] = useState("");
  const [error, setError] = useState<string | null>(null);

  function validate(addr: string): boolean {
    if (!/^0x[0-9a-fA-F]{40}$/.test(addr)) {
      setError("Invalid address — expected 0x + 40 hex characters")
      return false
    }
    setError(null)
    return true
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = value.trim();
    if (!trimmed || loading) return;
    if (validate(trimmed)) onSearch(trimmed);
  };

  return (
    <div className="w-full max-w-xl mx-auto">
      <form onSubmit={handleSubmit}>
        <div className="flex gap-2">
          <div className="relative flex-1">
            <input
              type="text"
              value={value}
              onChange={(e) => {
                setValue(e.target.value);
                if (error) setError(null);
              }}
              onBlur={() => value.trim() && validate(value.trim())}
              placeholder="Paste token address (0x...)"
              className={`w-full px-5 py-4 bg-gray-900/80 border rounded-xl
                          text-white placeholder-gray-500 font-mono text-sm
                          focus:outline-none focus:ring-1 transition-all
                          ${error
                            ? "border-red-500/60 focus:border-red-500 focus:ring-red-500"
                            : "border-gray-700 focus:border-purple-500 focus:ring-purple-500"}`}
              disabled={loading}
              spellCheck={false}
            />
          </div>
          <button
            type="submit"
            disabled={!value.trim() || loading}
            className="px-8 py-4 bg-gradient-to-r from-purple-600 to-cyan-500
                       hover:from-purple-500 hover:to-cyan-400
                       disabled:from-gray-700 disabled:to-gray-700 disabled:text-gray-500
                       rounded-xl font-semibold transition-all text-white whitespace-nowrap
                       shadow-lg shadow-purple-900/30"
          >
            {loading ? (
              <span className="flex items-center gap-2">
                <span className="inline-block w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                Scanning...
              </span>
            ) : (
              "🔍 Analyze"
            )}
          </button>
        </div>
      </form>

      {error && (
        <p className="mt-2 text-xs text-red-400 text-left pl-1">{error}</p>
      )}

      <div className="flex items-center justify-center gap-2 mt-4 flex-wrap">
        <span className="text-xs text-gray-600">Try:</span>
        {EXAMPLES.map((ex) => (
          <button
            key={ex.address}
            type="button"
            disabled={loading}
            onClick={() => {
              setValue(ex.address);
              setError(null);
              onSearch(ex.address);
            }}
            className="text-xs px-3 py-1.5 rounded-full border border-gray-700
                       text-gray-400 hover:text-purple-300 hover:border-purple-500/40
                       transition disabled:opacity-40"
          >
            🥞 {ex.symbol}
          </button>
        ))}
      </div>
    </div>
  );
}
