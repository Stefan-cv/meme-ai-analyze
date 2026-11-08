"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import SearchBar from "@/components/SearchBar";
import RiskWall from "@/components/RiskWall";
import StatsBar from "@/components/StatsBar";

const FEATURES = [
  {
    icon: "📡",
    title: "6-Dimension Engine",
    desc: "Contract safety, LP lock, holder concentration, creator history, liquidity & social signals — weighted scoring.",
  },
  {
    icon: "🤖",
    title: "AI Narrative Report",
    desc: "A plain-language risk story generated from on-chain data, with rule-based fallback when the LLM is offline.",
  },
  {
    icon: "⛓️",
    title: "On-chain Anchoring",
    desc: "Every report hash can be anchored to BSC via the RiskReportOracle contract — reports are tamper-proof.",
  },
];

export default function HomePage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);

  const handleSearch = (address: string) => {
    setLoading(true);
    router.push(`/scan/${address}`);
  };

  return (
    <div className="min-h-screen">
      {/* Hero */}
      <section className="flex flex-col items-center justify-center px-4 pt-20 pb-14">
        <div className="text-center mb-10 animate-fade-in">
          <div className="inline-flex items-center gap-2 text-xs px-3 py-1.5 rounded-full
                          border border-purple-500/30 bg-purple-500/10 text-purple-300 mb-6">
            <span className="w-1.5 h-1.5 rounded-full bg-green-400 animate-pulse" />
            Live on BSC mainnet · 15–30s per scan
          </div>
          <h1 className="text-5xl md:text-6xl font-bold mb-5 bg-gradient-to-r from-purple-400 via-fuchsia-300 to-cyan-400 bg-clip-text text-transparent">
            Meme AI Analyze
          </h1>
          <p className="text-lg text-gray-400 max-w-2xl mx-auto">
            Paste any BSC meme token address → get a{" "}
            <span className="text-white">six-dimension risk radar</span> +{" "}
            <span className="text-white">AI analysis report</span> in seconds.
            Your flashlight in the dark forest. 🔦
          </p>
        </div>

        <SearchBar onSearch={handleSearch} loading={loading} />
      </section>

      {/* How it works */}
      <section className="max-w-5xl mx-auto px-4 pb-14">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {FEATURES.map((f, i) => (
            <div
              key={f.title}
              className="gradient-card rounded-xl p-5 animate-fade-in"
              style={{ animationDelay: `${i * 90}ms` }}
            >
              <div className="text-2xl mb-3">{f.icon}</div>
              <h3 className="font-semibold text-white mb-2">{f.title}</h3>
              <p className="text-sm text-gray-400 leading-relaxed">{f.desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Stats */}
      <StatsBar />

      {/* Risk Wall */}
      <RiskWall />
    </div>
  );
}
