import type { ChessBoard, ChessColor, ChessMove, HistoryEntry } from "@/types/chess";
import { emptyBoard } from "@/types/chess";
import type { AiAnalysisResult, MultiPvLine } from "@/wasm/generated/wasm-contract";
import type { UseChessClock } from "@/hooks/useChessClock";
import type { PvSanEntry } from "@/utils/pvToSan";

export type { HistoryEntry } from "@/types/chess";

export type ChessGameMode = "human-vs-ai" | "human-vs-human" | "ai-vs-ai" | "analysis";
export type ChessResult = "white-wins" | "black-wins" | "draw" | null;
export type ChessDifficulty = "easy" | "medium" | "hard";
export type ChessSearchMode = "difficulty" | "time" | "depth";

export interface PendingPromotion {
  from: number;
  to: number;
  options: number[];
}

export interface ChessState {
  board: ChessBoard;
  currentPlayer: ChessColor;
  selectedSquare: number | null;
  validMoveSquares: number[];
  result: ChessResult;
  pendingPromotion: PendingPromotion | null;
  candidateMoves: ChessMove[];
  checkSquare: number | null;
  aiThinking: boolean;
  lastMove: { from: number; to: number } | null;
  boardBefore: ChessBoard | null;
  animateId: number;
}

export interface UseChessReturn {
  state: ChessState;
  history: HistoryEntry[];
  currentPly: number;
  isAtLatest: boolean;
  handleSquareClick: (index: number) => Promise<void>;
  playMove: (from: number, to: number, promotion?: number) => Promise<void>;
  restartGame: () => Promise<void>;
  choosePromotion: (promotionByte: number) => Promise<void>;
  cancelPromotion: () => void;
  jumpToPly: (ply: number) => void;
  clock: UseChessClock;
  gameStarted: boolean;
  analyze: (timeLimitMs: number) => Promise<AiAnalysisResult | null>;
  autoAnalyze: boolean;
  setAutoAnalyze: (on: boolean) => void;
  analyzeCurrentPosition: () => Promise<void>;
  analysisForPly: (ply: number) => AiAnalysisResult | null | undefined;
  exportPgn: () => string;
  loadPgn: (pgn: string) => Promise<boolean>;
  multiPv: MultiPvLine[];
  multiPvSan: PvSanEntry[][];
  multiPvDepth: number;
  multiPvThinking: boolean;
  currentFen: string;
  runMultiPv: (numLines: number, timeMs: number) => void;
  stopMultiPv: () => void;
}

export const initialState = (): ChessState => ({
  board: emptyBoard(),
  currentPlayer: "white",
  selectedSquare: null,
  validMoveSquares: [],
  result: null,
  pendingPromotion: null,
  candidateMoves: [],
  checkSquare: null,
  aiThinking: false,
  lastMove: null,
  boardBefore: null,
  animateId: 0,
});

export const pieceColor = (byte: number): ChessColor | null => {
  if (!byte) return null;
  return (byte & 0b11000000) === 0b01000000 ? "white" : "black";
};

export const toResult = (status: string): ChessResult => {
  if (status === "white-wins" || status === "black-wins" || status === "draw") {
    return status;
  }
  return null;
};

export const DIFFICULTY_MS: Record<ChessDifficulty, number> = {
  easy: 100,
  medium: 500,
  hard: 2000,
};

export const AI_DELAY_MS = 400;