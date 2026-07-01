// --- Function Contract ---
export interface WasmContract {
  validMovesChess: { args: []; return: string };
  initBoard: { args: []; return: number[] };
  makeMove: { args: [number, number, number?]; return: number[] };
  isCheckJS: { args: []; return: number };
  gameStatus: { args: []; return: string };
  aiMove: { args: [number]; return: string };
  aiMoveDepth: { args: [number]; return: string };
  aiAnalysis: { args: [number]; return: string };
  aiMultiPv: { args: [number, number]; return: string };
  loadBook: { args: [Uint8Array]; return: boolean };
  bookMoves: { args: []; return: string };
  fen: { args: []; return: string };
  san: { args: [number, number, number?]; return: string };
  applyPgn: { args: [string]; return: string };
}

export interface AiAnalysisResult {
  from: number;
  to: number;
  promotion?: number;
  score: number;
  depth: number;
  nodes: number;
  timeMs: number;
}

export interface PvMove {
  from: number;
  to: number;
  promotion?: number;
}

export interface MultiPvLine {
  moves: PvMove[];
  score: number;
  depth: number;
  nodes: number;
  timeMs: number;
}

export interface PgnHistoryEntry {
  san: string;
  from: number;
  to: number;
  promotion?: number;
  boardBefore: number[];
  boardAfter: number[];
  checkSquare: number;
  isCheckmate: boolean;
}

export interface BookMoveEntry {
  from: number;
  to: number;
  promotion?: number;
  weight: number;
  san: string;
}

export type WasmFunctionName = keyof WasmContract;

export type WasmEngine = {
  [K in WasmFunctionName]: (
    ...args: WasmContract[K]["args"]
  ) => Promise<WasmContract[K]["return"]>;
};