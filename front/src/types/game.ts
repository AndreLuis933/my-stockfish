export type Color = "white" | "black";
export type PieceType = "man" | "king";

export interface Piece {
  color: Color;
  type: PieceType;
}

export type Cell = Piece | null;
export type Board = Cell[][];

export interface Move {
  to: [number, number];
  captured: [number, number] | null;
}
