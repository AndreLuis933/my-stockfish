import { useEffect } from "react";
import type { ChessResult } from "@/pages/chess/Chess.types";

interface UseMultiPvControlParams {
  isAnalysisMode: boolean;
  analysisEnabled: boolean;
  maxLines: number;
  analysisTimeMs: number;
  result: ChessResult;
  isAtLatest: boolean;
  currentPly: number;
  runMultiPv: (numLines: number, timeMs: number) => void;
  stopMultiPv: () => void;
}

export const useMultiPvControl = ({
  isAnalysisMode,
  analysisEnabled,
  maxLines,
  analysisTimeMs,
  result,
  isAtLatest,
  currentPly,
  runMultiPv,
  stopMultiPv,
}: UseMultiPvControlParams): void => {
  useEffect(() => {
    if (!isAnalysisMode || !analysisEnabled) {
      stopMultiPv();
      return;
    }
    if (result && isAtLatest) {
      stopMultiPv();
      return;
    }
    runMultiPv(maxLines, analysisTimeMs);
    return () => stopMultiPv();
  }, [
    isAnalysisMode,
    analysisEnabled,
    maxLines,
    analysisTimeMs,
    result,
    isAtLatest,
    currentPly,
    runMultiPv,
    stopMultiPv,
  ]);
};