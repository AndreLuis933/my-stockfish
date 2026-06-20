import type { ChessBoard, ChessColor, ChessPieceType } from "@/types/chess";
import { getPiece } from "@/types/chess";

const FILES = "abcdefgh";

export const squareName = (index: number): string => {
  const file = FILES[index % 8];
  const rank = Math.floor(index / 8);
  return `${file}${8 - rank}`;
};

const PIECE_LETTERS: Record<ChessPieceType, string> = {
  king: "K",
  queen: "Q",
  rook: "R",
  bishop: "B",
  knight: "N",
  pawn: "",
};

const PROMOTION_LETTERS: Record<number, string> = {
  0b00000010: "N",
  0b00000100: "B",
  0b00001000: "R",
  0b00010000: "Q",
};

const colorBits = (color: ChessColor): number =>
  color === "white" ? 0b01000000 : 0b10000000;

const isPathClear = (
  board: ChessBoard,
  from: number,
  dr: number,
  dc: number,
  isTargetEnemy: boolean,
  isTargetEmpty: boolean,
): boolean => {
  const steps = Math.max(Math.abs(dr), Math.abs(dc));
  const stepR = Math.sign(dr);
  const stepC = Math.sign(dc);
  const fromR = Math.floor(from / 8);
  const fromC = from % 8;

  for (let i = 1; i < steps; i++) {
    const idx = (fromR + stepR * i) * 8 + (fromC + stepC * i);
    if (getPiece(board, idx)) return false;
  }
  return isTargetEmpty || isTargetEnemy;
};

const canPieceReach = (
  board: ChessBoard,
  from: number,
  to: number,
  pieceType: ChessPieceType,
  color: ChessColor,
): boolean => {
  if (from === to) return false;
  const dr = Math.floor(to / 8) - Math.floor(from / 8);
  const dc = (to % 8) - (from % 8);
  const targetPiece = getPiece(board, to);
  const isTargetEnemy = targetPiece !== null && targetPiece.color !== color;
  const isTargetEmpty = targetPiece === null;

  switch (pieceType) {
    case "knight":
      return (
        (Math.abs(dr) === 2 && Math.abs(dc) === 1) ||
        (Math.abs(dr) === 1 && Math.abs(dc) === 2)
      );
    case "bishop":
      if (Math.abs(dr) !== Math.abs(dc) || dr === 0) return false;
      return isPathClear(board, from, dr, dc, isTargetEnemy, isTargetEmpty);
    case "rook":
      if (dr !== 0 && dc !== 0) return false;
      return isPathClear(board, from, dr, dc, isTargetEnemy, isTargetEmpty);
    case "queen": {
      const isDiagonal = Math.abs(dr) === Math.abs(dc) && dr !== 0;
      const isStraight = dr === 0 || dc === 0;
      if (!isDiagonal && !isStraight) return false;
      return isPathClear(board, from, dr, dc, isTargetEnemy, isTargetEmpty);
    }
    case "king":
      return Math.abs(dr) <= 1 && Math.abs(dc) <= 1;
    case "pawn":
      return false;
  }
};

export interface MoveData {
  from: number;
  to: number;
  promotion?: number;
}

export const toSan = (
  boardBefore: ChessBoard,
  move: MoveData,
  color: ChessColor,
  checkSquareAfter: number | null,
  isCheckmate: boolean,
): string => {
  const piece = getPiece(boardBefore, move.from);
  if (!piece) return squareName(move.to);

  if (piece.type === "king" && Math.abs((move.from % 8) - (move.to % 8)) === 2) {
    const kingside = (move.to % 8) === 6;
    const base = kingside ? "O-O" : "O-O-O";
    return appendCheckSuffix(base, checkSquareAfter, isCheckmate);
  }

  const targetPiece = getPiece(boardBefore, move.to);
  const isCapture = targetPiece !== null;
  const isEnPassant =
    piece.type === "pawn" &&
    (move.from % 8) !== (move.to % 8) &&
    !isCapture;

  const letter = PIECE_LETTERS[piece.type];

  let disambiguation = "";
  if (piece.type !== "pawn" && piece.type !== "king") {
    const candidates: number[] = [];
    for (let i = 0; i < 64; i++) {
      if (i === move.from) continue;
      const p = getPiece(boardBefore, i);
      if (!p || p.type !== piece.type || p.color !== color) continue;
      if (canPieceReach(boardBefore, i, move.to, piece.type, color)) {
        candidates.push(i);
      }
    }

    if (candidates.length > 0) {
      const fromFile = move.from % 8;
      const fromRank = Math.floor(move.from / 8);
      const sameFile = candidates.some((c) => c % 8 === fromFile);
      const sameRank = candidates.some(
        (c) => Math.floor(c / 8) === fromRank,
      );

      if (!sameFile) {
        disambiguation = FILES[fromFile];
      } else if (!sameRank) {
        disambiguation = String(8 - fromRank);
      } else {
        disambiguation = squareName(move.from);
      }
    }
  }

  let san: string;
  if (piece.type === "pawn") {
    if (isCapture || isEnPassant) {
      san = `${FILES[move.from % 8]}x${squareName(move.to)}`;
    } else {
      san = squareName(move.to);
    }
    if (move.promotion && PROMOTION_LETTERS[move.promotion & 0b00111111]) {
      san += `=${PROMOTION_LETTERS[move.promotion & 0b00111111]}`;
    }
  } else {
    san = `${letter}${disambiguation}${isCapture ? "x" : ""}${squareName(move.to)}`;
  }

  return appendCheckSuffix(san, checkSquareAfter, isCheckmate);
};

const appendCheckSuffix = (
  san: string,
  checkSquare: number | null,
  isCheckmate: boolean,
): string => {
  if (isCheckmate) return `${san}#`;
  if (checkSquare !== null) return `${san}+`;
  return san;
};

export const promoByte = (
  color: ChessColor,
  type: ChessPieceType,
): number => colorBits(color) | promoTypeBits(type);

const promoTypeBits = (type: ChessPieceType): number => {
  const map: Record<ChessPieceType, number> = {
    pawn: 0b00000001,
    knight: 0b00000010,
    bishop: 0b00000100,
    rook: 0b00001000,
    queen: 0b00010000,
    king: 0b00100000,
  };
  return map[type];
};