import { useCallback, useMemo, useRef, useState } from "react";
import type { ChessBoard, ChessColor } from "@/types/chess";
import type { MultiPvLine, WasmEngine } from "@/wasm/generated/wasm-contract";
import { pvToSan, type PvSanEntry } from "@/utils/pvToSan";

export interface UseMultiPvReturn {
  multiPv: MultiPvLine[];
  multiPvSan: PvSanEntry[][];
  multiPvDepth: number;
  multiPvThinking: boolean;
  runMultiPv: (numLines: number, timeMs: number) => void;
  stopMultiPv: () => void;
}

export const useMultiPv = (
  engine: WasmEngine | null,
  board: ChessBoard,
  currentPlayer: ChessColor,
): UseMultiPvReturn => {
  const [multiPv, setMultiPv] = useState<MultiPvLine[]>([]);
  const [multiPvThinking, setMultiPvThinking] = useState(false);
  const multiPvRef = useRef<{ cancelled: boolean } | null>(null);

  const multiPvSan = useMemo<PvSanEntry[][]>(() => {
    if (multiPv.length === 0) return [];
    const boardCopy = board.slice();
    const startColor: ChessColor = currentPlayer;
    return multiPv.map((line) => pvToSan(boardCopy, line.moves, startColor));
  }, [multiPv, board, currentPlayer]);

  const multiPvDepth = multiPv.length > 0 ? Math.max(...multiPv.map((l) => l.depth)) : 0;

  const runMultiPv = useCallback(
    (numLines: number, timeMs: number) => {
      if (!engine) return;
      if (multiPvRef.current) multiPvRef.current.cancelled = true;
      const token = { cancelled: false };
      multiPvRef.current = token;
      const currentEngine = engine;

      setMultiPvThinking(true);
      setMultiPv([]);

      async function loop() {
        while (!token.cancelled) {
          const json = await currentEngine.aiMultiPv(timeMs, numLines);
          if (token.cancelled) break;
          const lines = JSON.parse(json) as MultiPvLine[];
          setMultiPv(lines);
        }
        if (!token.cancelled) {
          setMultiPvThinking(false);
        }
      }
      loop();
    },
    [engine],
  );

  const stopMultiPv = useCallback(() => {
    if (multiPvRef.current) {
      multiPvRef.current.cancelled = true;
      multiPvRef.current = null;
    }
    setMultiPvThinking(false);
  }, []);

  return {
    multiPv,
    multiPvSan,
    multiPvDepth,
    multiPvThinking,
    runMultiPv,
    stopMultiPv,
  };
};