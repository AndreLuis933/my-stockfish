import { useEffect } from "react";
import type { ChessColor, ChessMove } from "@/types/chess";
import type { WasmEngine } from "@/wasm/generated/wasm-contract";
import type {
  ChessDifficulty,
  ChessGameMode,
  ChessResult,
  ChessSearchMode,
  ChessState,
} from "@/pages/chess/Chess.types";
import { AI_DELAY_MS, DIFFICULTY_MS } from "@/pages/chess/Chess.types";

interface UseChessAiParams {
  engine: WasmEngine | null;
  mode: ChessGameMode;
  aiColor: ChessColor;
  difficulty: ChessDifficulty;
  searchMode: ChessSearchMode;
  customTimeMs: number;
  customDepth: number;
  currentPlayer: ChessColor;
  result: ChessResult;
  isAtLatest: boolean;
  applyMove: (
    engine: WasmEngine,
    from: number,
    to: number,
    promotion: number,
    moverColor: ChessColor,
  ) => Promise<void>;
  setState: React.Dispatch<React.SetStateAction<ChessState>>;
}

export const useChessAi = ({
  engine,
  mode,
  aiColor,
  difficulty,
  searchMode,
  customTimeMs,
  customDepth,
  currentPlayer,
  result,
  isAtLatest,
  applyMove,
  setState,
}: UseChessAiParams): void => {
  useEffect(() => {
    if (!engine || result !== null) return;
    if (mode === "human-vs-human" || mode === "analysis") return;
    if (mode === "human-vs-ai" && currentPlayer !== aiColor) return;
    if (!isAtLatest) return;

    const currentEngine = engine;
    const moverColor = currentPlayer;
    let cancelled = false;

    const timer = setTimeout(async () => {
      setState((p) => ({ ...p, aiThinking: true }));

      let moveJson: string;
      if (searchMode === "depth") {
        moveJson = await currentEngine.aiMoveDepth(customDepth);
      } else if (searchMode === "time") {
        moveJson = await currentEngine.aiMove(customTimeMs);
      } else {
        moveJson = await currentEngine.aiMove(DIFFICULTY_MS[difficulty]);
      }
      const aiMove: ChessMove = JSON.parse(moveJson);

      if (cancelled) return;

      await applyMove(
        currentEngine,
        aiMove.from,
        aiMove.to,
        aiMove.promotion ?? 0,
        moverColor,
      );

      if (!cancelled) {
        setState((p) => ({ ...p, aiThinking: false }));
      }
    }, AI_DELAY_MS);

    return () => {
      cancelled = true;
      clearTimeout(timer);
      setState((p) => ({ ...p, aiThinking: false }));
    };
  }, [
    engine,
    mode,
    aiColor,
    difficulty,
    searchMode,
    customTimeMs,
    customDepth,
    currentPlayer,
    result,
    isAtLatest,
    applyMove,
    setState,
  ]);
};