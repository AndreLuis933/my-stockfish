import { describe, it, expect } from "vitest";
import type { Board } from "@/types/game";
import { applyMove, checkResult, computeTurnState, initBoard, validMoves } from "@/utils/gameEngine";

const emptyBoard = (): Board => Array.from({ length: 8 }, () => Array(8).fill(null));

describe("initBoard", () => {
  it("places 12 white pieces on rows 0–2", () => {
    const board = initBoard();
    let count = 0;
    for (let r = 0; r < 3; r++)
      for (let c = 0; c < 8; c++)
        if (board[r][c]?.color === "white") count++;
    expect(count).toBe(12);
  });

  it("places 12 black pieces on rows 5–7", () => {
    const board = initBoard();
    let count = 0;
    for (let r = 5; r < 8; r++)
      for (let c = 0; c < 8; c++)
        if (board[r][c]?.color === "black") count++;
    expect(count).toBe(12);
  });

  it("only places pieces on dark squares (odd sum)", () => {
    const board = initBoard();
    for (let r = 0; r < 8; r++)
      for (let c = 0; c < 8; c++)
        if ((r + c) % 2 === 0) expect(board[r][c]).toBeNull();
  });

  it("rows 3–4 are empty", () => {
    const board = initBoard();
    for (let c = 0; c < 8; c++) {
      expect(board[3][c]).toBeNull();
      expect(board[4][c]).toBeNull();
    }
  });
});

describe("computeTurnState", () => {
  it("initial board: white has selectable pieces and no forced captures", () => {
    const board = initBoard();
    const ts = computeTurnState(board, "white");
    expect(ts.globalMax).toBe(0);
    expect(ts.selectable.length).toBeGreaterThan(0);
  });

  it("forces the only capture when available", () => {
    const board = emptyBoard();
    board[3][2] = { color: "white", type: "man" };
    board[4][3] = { color: "black", type: "man" };
    const ts = computeTurnState(board, "white");
    expect(ts.globalMax).toBe(1);
    expect(ts.selectable).toEqual([[3, 2]]);
  });

  it("selects only the piece with the longest capture chain", () => {
    const board = emptyBoard();
    // [0,1] chains two captures: jumps [1,2] → lands [2,3] → jumps [3,2] → lands [4,1]
    board[0][1] = { color: "white", type: "man" };
    board[1][2] = { color: "black", type: "man" };
    board[3][2] = { color: "black", type: "man" };
    // [6,7] can only capture one piece ([5,6] → lands [4,5], no continuation)
    board[6][7] = { color: "white", type: "man" };
    board[5][6] = { color: "black", type: "man" };
    const ts = computeTurnState(board, "white");
    expect(ts.globalMax).toBe(2);
    expect(ts.selectable).toEqual([[0, 1]]);
  });

  it("returns empty selectable when the color has no pieces", () => {
    const ts = computeTurnState(emptyBoard(), "white");
    expect(ts.selectable).toEqual([]);
    expect(ts.globalMax).toBe(0);
  });
});

describe("validMoves", () => {
  it("returns forward diagonal moves for a white man", () => {
    const board = emptyBoard();
    board[3][3] = { color: "white", type: "man" };
    const moves = validMoves(3, 3, board, 0);
    const destinations = moves.map((m) => m.to);
    expect(destinations).toContainEqual([4, 2]);
    expect(destinations).toContainEqual([4, 4]);
    expect(moves.every((m) => m.captured.length === 0)).toBe(true);
  });

  it("returns only capture moves when globalMax > 0", () => {
    const board = emptyBoard();
    board[3][2] = { color: "white", type: "man" };
    board[4][3] = { color: "black", type: "man" };
    const ts = computeTurnState(board, "white");
    const moves = validMoves(3, 2, board, ts.globalMax);
    expect(moves.length).toBe(1);
    expect(moves[0].to).toEqual([5, 4]);
    expect(moves[0].captured).toEqual([[4, 3]]);
  });

  it("king can move in all four diagonal directions", () => {
    const board = emptyBoard();
    board[4][4] = { color: "white", type: "king" };
    const moves = validMoves(4, 4, board, 0);
    const destinations = moves.map((m) => m.to);
    expect(destinations.some(([r, c]) => r < 4 && c < 4)).toBe(true);
    expect(destinations.some(([r, c]) => r < 4 && c > 4)).toBe(true);
    expect(destinations.some(([r, c]) => r > 4 && c < 4)).toBe(true);
    expect(destinations.some(([r, c]) => r > 4 && c > 4)).toBe(true);
  });
});

describe("applyMove", () => {
  it("moves the piece from source to destination", () => {
    const board = emptyBoard();
    board[3][2] = { color: "white", type: "man" };
    const next = applyMove([3, 2], { to: [4, 3], captured: [] }, board);
    expect(next[3][2]).toBeNull();
    expect(next[4][3]).toEqual({ color: "white", type: "man" });
  });

  it("removes captured pieces", () => {
    const board = emptyBoard();
    board[3][2] = { color: "white", type: "man" };
    board[4][3] = { color: "black", type: "man" };
    const next = applyMove([3, 2], { to: [5, 4], captured: [[4, 3]] }, board);
    expect(next[4][3]).toBeNull();
    expect(next[5][4]).toEqual({ color: "white", type: "man" });
  });

  it("promotes white man to king at row 7", () => {
    const board = emptyBoard();
    board[6][1] = { color: "white", type: "man" };
    const next = applyMove([6, 1], { to: [7, 2], captured: [] }, board);
    expect(next[7][2]).toEqual({ color: "white", type: "king" });
  });

  it("promotes black man to king at row 0", () => {
    const board = emptyBoard();
    board[1][2] = { color: "black", type: "man" };
    const next = applyMove([1, 2], { to: [0, 3], captured: [] }, board);
    expect(next[0][3]).toEqual({ color: "black", type: "king" });
  });

  it("does not mutate the original board", () => {
    const board = emptyBoard();
    board[3][2] = { color: "white", type: "man" };
    applyMove([3, 2], { to: [4, 3], captured: [] }, board);
    expect(board[3][2]).toEqual({ color: "white", type: "man" });
    expect(board[4][3]).toBeNull();
  });
});

describe("checkResult", () => {
  it("returns null when the game is ongoing", () => {
    const board = initBoard();
    const ts = computeTurnState(board, "white");
    expect(checkResult(ts, "white", 0)).toBeNull();
  });

  it("returns black-wins when white has no moves", () => {
    expect(checkResult({ selectable: [], globalMax: 0 }, "white", 0)).toBe("black-wins");
  });

  it("returns white-wins when black has no moves", () => {
    expect(checkResult({ selectable: [], globalMax: 0 }, "black", 0)).toBe("white-wins");
  });

  it("returns draw at exactly 40 moves without capture", () => {
    const board = initBoard();
    const ts = computeTurnState(board, "white");
    expect(checkResult(ts, "white", 40)).toBe("draw");
  });

  it("does not draw at 39 moves without capture", () => {
    const board = initBoard();
    const ts = computeTurnState(board, "white");
    expect(checkResult(ts, "white", 39)).toBeNull();
  });
});
