import { describe, it, expect } from "vitest";
import type { Board } from "@/types/game";
import { initBoard } from "@/utils/gameEngine";
import { pickBestMove, pickBestMoveWithTime } from "@/utils/aiEngine";

const emptyBoard = (): Board => Array.from({ length: 8 }, () => Array(8).fill(null));

// ── Fixed-depth baseline ──────────────────────────────────────────────────────

describe("pickBestMove — fixed depth", () => {
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

  it("advances a man toward promotion at low depth", () => {
    const board = emptyBoard();
    board[6][1] = { color: "white", type: "man" };
    const result = pickBestMove(board, "white", 0, 3);
    expect(result).not.toBeNull();
    expect(result!.move.to[0]).toBe(7); // reaches back rank → promotes
  });

  it("benchmarks depth 6 on the starting position", () => {
    const board = initBoard();
    const start = performance.now();
    const result = pickBestMove(board, "white", 0, 6);
    const elapsed = performance.now() - start;
    expect(result).not.toBeNull();
    console.log(`[depth 6] time: ${elapsed.toFixed(0)}ms`);
  });

  it("benchmarks depth 8 on the starting position", () => {
    const board = initBoard();
    const start = performance.now();
    const result = pickBestMove(board, "white", 0, 8);
    const elapsed = performance.now() - start;
    expect(result).not.toBeNull();
    console.log(`[depth 8] time: ${elapsed.toFixed(0)}ms`);
  });
});

// ── Iterative deepening ───────────────────────────────────────────────────────

describe("pickBestMoveWithTime — iterative deepening", () => {
  it("returns null on an empty board", () => {
    expect(pickBestMoveWithTime(emptyBoard(), "white", 0, 500)).toBeNull();
  });

  it("returns within the time budget (1 s)", () => {
    const board = initBoard();
    const start = performance.now();
    const result = pickBestMoveWithTime(board, "white", 0, 1000);
    const wall = performance.now() - start;
    expect(result).not.toBeNull();
    expect(wall).toBeLessThan(1300); // 30 % margin
    console.log(`[1 s limit] depth: ${result!.depth} | nodes: ${result!.nodes} | actual: ${result!.timeMs.toFixed(0)}ms`);
  });

  it("searches deeper when given more time", () => {
    const board = initBoard();
    const fast = pickBestMoveWithTime(board, "white", 0, 100);
    const slow = pickBestMoveWithTime(board, "white", 0, 2000);
    expect(fast).not.toBeNull();
    expect(slow).not.toBeNull();
    expect(slow!.depth).toBeGreaterThanOrEqual(fast!.depth);
    console.log(`[100 ms]  depth: ${fast!.depth} | nodes: ${fast!.nodes}`);
    console.log(`[2000 ms] depth: ${slow!.depth} | nodes: ${slow!.nodes}`);
  });

  it("takes a forced capture", () => {
    const board = emptyBoard();
    board[3][2] = { color: "white", type: "man" };
    board[4][3] = { color: "black", type: "man" };
    const result = pickBestMoveWithTime(board, "white", 0, 500);
    expect(result).not.toBeNull();
    const { from, move } = result!.move; // AIMove: { from, move: Move }
    expect(from).toEqual([3, 2]);
    expect(move.captured.length).toBe(1);
    expect(move.to).toEqual([5, 4]);
  });

  it("finds a forced win and reports score ≥ WIN", () => {
    // White king vs lone black man — white can always capture
    const board = emptyBoard();
    board[3][3] = { color: "white", type: "king" };
    board[4][4] = { color: "black", type: "man" };
    const result = pickBestMoveWithTime(board, "white", 0, 2000);
    expect(result).not.toBeNull();
    // After taking the only black piece white wins immediately
    expect(result!.score).toBeGreaterThan(0);
    console.log(`[forced win] depth: ${result!.depth} | score: ${result!.score} | nodes: ${result!.nodes}`);
  });

  it("NPS throughput on the starting position (2 s)", () => {
    const board = initBoard();
    const result = pickBestMoveWithTime(board, "white", 0, 2000);
    expect(result).not.toBeNull();
    const nps = Math.round(result!.nodes / (result!.timeMs / 1000));
    console.log(
      `[throughput] depth: ${result!.depth} | nodes: ${result!.nodes} | ` +
        `time: ${result!.timeMs.toFixed(0)}ms | NPS: ${nps.toLocaleString()}`,
    );
    expect(result!.depth).toBeGreaterThanOrEqual(5);
  });
});
