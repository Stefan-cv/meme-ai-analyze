import { ImageResponse } from "next/og";
import { NextRequest } from "next/server";

export const runtime = "edge";

export async function GET(req: NextRequest) {
  const tokenAddress = req.nextUrl.pathname.split("/").pop() || "";

  return new ImageResponse(
    (
      <div
        style={{
          height: "100%",
          width: "100%",
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          justifyContent: "center",
          background: "linear-gradient(135deg, #0a0a0f 0%, #1a1040 100%)",
          fontFamily: "Inter",
        }}
      >
        <div
          style={{
            fontSize: 48,
            fontWeight: 700,
            background: "linear-gradient(to right, #a855f7, #22d3ee)",
            backgroundClip: "text",
            color: "transparent",
            marginBottom: 20,
          }}
        >
          Meme AI Analyze
        </div>
        <div
          style={{
            fontSize: 20,
            color: "#94a3b8",
            maxWidth: 600,
            textAlign: "center",
          }}
        >
          Risk report for {tokenAddress.slice(0, 10)}...
        </div>
        <div
          style={{
            fontSize: 16,
            color: "#64748b",
            marginTop: 40,
          }}
        >
          scan → analyze → protect
        </div>
      </div>
    ),
    {
      width: 1200,
      height: 630,
    }
  );
}
