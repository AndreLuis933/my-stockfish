// --- Function Contract ---
export interface WasmContract {
  validMovesChess: { args: []; return: string };
  initBoard: { args: []; return: number[] };
  makeMove: { args: [number, number, number?]; return: number[] };
  isCheckJS: { args: []; return: number };
  gameStatus: { args: []; return: string };
}
;
export type WasmFunctionName = keyof WasmContract;

export type WasmEngine = {
  [K in WasmFunctionName]: (
    ...args: WasmContract[K]["args"]
  ) => Promise<WasmContract[K]["return"]>;
};

export type WasmResult<T> =
  | { ok: true; value: T }
  | { ok: false; error: string };
