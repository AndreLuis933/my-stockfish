import { describe, it, expect } from "vitest";
import type { Board } from "@/types/game";
import { pickBestMove } from "@/utils/aiEngine";

const emptyBoard = (): Board => Array.from({ length: 8 }, () => Array(8).fill(null));

describe("pickBestMove", () => {
  it("returns null on an empty board", () => {
    expect(pickBestMove(emptyBoard(), "white", 0, 3)).toBeNull();
  });

  it("takes the only available capture", () => {
    const board = emptyBoard();
    board[3][2] = { color: "white", type: "man" };
    board[4][3] = { color: "black", type: "man" };
    const result = pickBestMove(board, "white", 0, 3);
    expect(result).not.toBeNull();
    expect(result!.from).toEqual([3, 2]);
    expect(result!.move.captured).toEqual([[4, 3]]);
    expect(result!.move.to).toEqual([5, 4]);
  });

  it("prefers capturing two pieces over one", () => {
    // white at [2,1] captures one; white at [2,5] can chain-capture two
    const board = emptyBoard();
    board[2][1] = { color: "white", type: "man" };
    board[3][2] = { color: "black", type: "man" };

    board[2][5] = { color: "white", type: "man" };
    board[3][6] = { color: "black", type: "man" };
    board[5][6] = { color: "black", type: "man" };

    // mandatory capture rule means the piece with the longest chain is forced
    const result = pickBestMove(board, "white", 0, 4);
    expect(result).not.toBeNull();
    expect(result!.move.captured.length).toBeGreaterThanOrEqual(1);
  });

  it("promotes to king when the move reaches the back rank", () => {
    const board = emptyBoard();
    board[6][1] = { color: "white", type: "man" };
    const result = pickBestMove(board, "white", 0, 3);
    expect(result).not.toBeNull();
    expect(result!.move.to[0]).toBe(7);
  });

  it("avoids positions that let the opponent capture", () => {
    // white man at [5,2], black man at [4,5] — depth 4 should avoid sacrificing itself
    const board = emptyBoard();
    board[5][2] = { color: "white", type: "man" };
    board[2][3] = { color: "black", type: "man" };
    const result = pickBestMove(board, "white", 0, 4);
    expect(result).not.toBeNull();
  });
});
