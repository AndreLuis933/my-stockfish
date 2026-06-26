import { useCallback, useEffect, useState } from "react";
import type { HistoryEntry } from "@/types/chess";
import type { AiAnalysisResult, WasmEngine } from "@/wasm/generated/wasm-contract";

export interface UseChessAnalysisReturn {
  analyze: (timeLimitMs: number) => Promise<AiAnalysisResult | null>;
  autoAnalyze: boolean;
  setAutoAnalyze: (on: boolean) => void;
  analyzeCurrentPosition: () => Promise<void>;
  analysisForPly: (ply: number) => AiAnalysisResult | null | undefined;
}

interface UseChessAnalysisParams {
  engine: WasmEngine | null;
  history: HistoryEntry[];
  setHistory: React.Dispatch<React.SetStateAction<HistoryEntry[]>>;
  currentPly: number;
  isAtLatest: boolean;
}

export const useChessAnalysis = ({
  engine,
  history,
  setHistory,
  currentPly,
  isAtLatest,
}: UseChessAnalysisParams): UseChessAnalysisReturn => {
  const [autoAnalyze, setAutoAnalyze] = useState(false);

  const analyze = useCallback(
    async (timeLimitMs: number): Promise<AiAnalysisResult | null> => {
      if (!engine) return null;
      const json = await engine.aiAnalysis(timeLimitMs);
      return JSON.parse(json) as AiAnalysisResult;
    },
    [engine],
  );

  useEffect(() => {
    if (!engine || !autoAnalyze) return;
    if (history.length === 0) return;
    if (!isAtLatest) return;

    const latestPly = history.length;
    const latestEntry = history[latestPly - 1];
    if (latestEntry.analysis) return;
    if (latestEntry.isCheckmate) return;

    let cancelled = false;

    const timer = setTimeout(async () => {
      const result = await analyze(500);
      if (cancelled) return;
      setHistory((h) => {
        if (h.length < latestPly) return h;
        const updated = h.slice();
        updated[latestPly - 1] = { ...updated[latestPly - 1], analysis: result };
        return updated;
      });
    }, 200);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [engine, autoAnalyze, history, isAtLatest, analyze, setHistory]);

  const analysisForPly = useCallback(
    (ply: number): AiAnalysisResult | null | undefined => {
      if (ply <= 0 || ply > history.length) return undefined;
      return history[ply - 1].analysis;
    },
    [history],
  );

  const analyzeCurrentPosition = useCallback(async () => {
    if (!engine) return;
    if (currentPly <= 0 || currentPly > history.length) return;

    const ply = currentPly;
    const result = await analyze(1000);
    if (!result) return;

    setHistory((h) => {
      if (ply > h.length) return h;
      const updated = h.slice();
      updated[ply - 1] = { ...updated[ply - 1], analysis: result };
      return updated;
    });
  }, [engine, analyze, currentPly, history.length, setHistory]);

  return {
    analyze,
    autoAnalyze,
    setAutoAnalyze,
    analyzeCurrentPosition,
    analysisForPly,
  };
};