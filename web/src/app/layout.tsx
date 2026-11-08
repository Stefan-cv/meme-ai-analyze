import type { Metadata } from "next";
import Link from "next/link";
import "./globals.css";

export const metadata: Metadata = {
  title: "Meme AI Analyze — Meme Token Risk Scanner",
  description:
    "Input any Meme token address → get a six-dimension risk radar + AI analysis report in 15 seconds. Know if it's safe before you ape.",
  icons: {
    icon: "data:image/svg+xml,<svg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 100 100%22><text y=%22.9em%22 font-size=%2290%22>🛡️</text></svg>",
  },
  openGraph: {
    title: "Meme AI Analyze",
    description: "The flashlight in the dark forest of Meme tokens.",
    type: "website",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="min-h-screen gradient-hero antialiased">
        <nav className="sticky top-0 z-50 border-b border-gray-800/60 bg-[#0a0a0f]/80 backdrop-blur-md">
          <div className="max-w-5xl mx-auto px-4 h-14 flex items-center justify-between">
            <Link href="/" className="flex items-center gap-2 group">
              <span className="text-xl">🛡️</span>
              <span className="font-bold text-white tracking-wide group-hover:text-purple-300 transition">
                Meme AI Analyze
              </span>
            </Link>
            <div className="flex items-center gap-6 text-sm">
              <Link
                href="/"
                className="text-gray-400 hover:text-white transition"
              >
                Scanner
              </Link>
              <Link
                href="/history"
                className="text-gray-400 hover:text-white transition"
              >
                History
              </Link>
              <span
                className="hidden sm:inline-flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-full
                             border border-purple-500/30 bg-purple-500/10 text-purple-300"
              >
                <span className="w-1.5 h-1.5 rounded-full bg-green-400 animate-pulse" />
                BSC Live
              </span>
            </div>
          </div>
        </nav>

        <main className="min-h-screen">{children}</main>

        <footer className="border-t border-gray-800/60 py-8 text-center text-xs text-gray-600">
          <p>
            Go · go-zero · PostgreSQL · Redis · Next.js 14 · TypeScript ·
            Foundry — on-chain risk engine for BSC meme tokens
          </p>
          <p className="mt-2">
            ⚠️ Risk scores are for reference only. Not financial advice.
          </p>
        </footer>
      </body>
    </html>
  );
}
