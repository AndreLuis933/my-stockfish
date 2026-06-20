import type { ChessBoard as ChessBoardState, ChessPiece } from "@/types/chess";
import { getPiece } from "@/types/chess";
import { pieceImageUrl } from "@/utils/chessAssets";
import styles from "./ChessBoard.module.css";

interface ChessBoardProps {
  board: ChessBoardState;
  selectedSquare?: number | null;
  validMoveSquares?: number[];
  onSquareClick?: (index: number) => Promise<void>;
  flipped?: boolean;
  checkSquare?: number | null;
  lastMove?: { from: number; to: number } | null;
  boardBefore?: ChessBoardState | null;
  animateId?: number;
  arrow?: { from: number; to: number } | null;
}

const INDICES = Array.from({ length: 8 }, (_, i) => i);
const FILES = "abcdefgh";

interface AnimInfo {
  from: number;
  to: number;
  rookFrom: number;
  rookTo: number;
  capturedPiece: ChessPiece | null;
  capturedAt: number;
}

const computeSlideOffset = (
  from: number,
  to: number,
  flipped: boolean,
): { dx: string; dy: string } => {
  const fromRow = Math.floor(from / 8);
  const fromCol = from % 8;
  const toRow = Math.floor(to / 8);
  const toCol = to % 8;
  const sign = flipped ? -1 : 1;
  const dx = sign * (fromCol - toCol);
  const dy = sign * (toRow - fromRow);
  return { dx: `${dx * 100}%`, dy: `${dy * 100}%` };
};

const visualRowCol = (
  index: number,
  flipped: boolean,
): { vRow: number; vCol: number } => {
  const r = Math.floor(index / 8);
  const c = index % 8;
  return {
    vRow: flipped ? r : 7 - r,
    vCol: flipped ? 7 - c : c,
  };
};

const computeAnim = (
  board: ChessBoardState,
  boardBefore: ChessBoardState,
  lastMove: { from: number; to: number },
): AnimInfo => {
  const piece = getPiece(board, lastMove.to);
  const isKing = piece?.type === "king";
  const fromCol = lastMove.from % 8;
  const toCol = lastMove.to % 8;
  const isCastling = isKing && Math.abs(fromCol - toCol) === 2;

  let capturedPiece: ChessPiece | null = null;
  let capturedAt = -1;
  for (let i = 0; i < 64; i++) {
    if (i === lastMove.from) continue;
    if (boardBefore[i] !== 0 && board[i] === 0) {
      capturedPiece = getPiece(boardBefore, i);
      capturedAt = i;
      break;
    }
  }

  let rookFrom = -1;
  let rookTo = -1;
  if (isCastling) {
    const row = Math.floor(lastMove.from / 8);
    if (toCol === 6) {
      rookFrom = row * 8 + 7;
      rookTo = row * 8 + 5;
    } else {
      rookFrom = row * 8 + 0;
      rookTo = row * 8 + 3;
    }
  }

  return { from: lastMove.from, to: lastMove.to, rookFrom, rookTo, capturedPiece, capturedAt };
};

export const ChessBoard = ({
  board,
  selectedSquare = null,
  validMoveSquares = [],
  onSquareClick,
  flipped = false,
  checkSquare = null,
  lastMove = null,
  boardBefore = null,
  animateId = 0,
  arrow = null,
}: ChessBoardProps) => {
  const shouldAnimate = animateId > 0 && !!lastMove && !!boardBefore;

  const animInfo =
    shouldAnimate && lastMove && boardBefore
      ? computeAnim(board, boardBefore, lastMove)
      : null;

  const rowIndices = flipped ? INDICES : [...INDICES].reverse();
  const colIndices = flipped ? [...INDICES].reverse() : INDICES;

  const ghostPos =
    animInfo && animInfo.capturedPiece && animInfo.capturedAt >= 0
      ? visualRowCol(animInfo.capturedAt, flipped)
      : null;

  return (
    <div className={styles.boardWrapper}>
      <div className={styles.board}>
        {rowIndices.map((r, vRow) =>
          colIndices.map((c, vCol) => {
            const index = r * 8 + c;
            const piece = getPiece(board, index);
            const isDark = (r + c) % 2 !== 0;
            const isSelected = selectedSquare === index;
            const isValidTarget = validMoveSquares.includes(index);
            const isInCheck = checkSquare === index;
            const isLastFrom = lastMove?.from === index;
            const isLastTo = lastMove?.to === index;

            const isSliding =
              animInfo !== null &&
              (index === animInfo.to || index === animInfo.rookTo);
            const slideFrom =
              animInfo !== null && index === animInfo.to
                ? animInfo.from
                : animInfo !== null && index === animInfo.rookTo
                  ? animInfo.rookFrom
                  : -1;

            const slideOffset =
              isSliding && slideFrom >= 0
                ? computeSlideOffset(slideFrom, index, flipped)
                : null;

            const squareClass = [
              styles.square,
              isDark ? styles.dark : styles.light,
              isSelected ? styles.selected : "",
              isInCheck ? styles.check : "",
              (isLastFrom || isLastTo) && !isSelected ? styles.lastMove : "",
            ]
              .filter(Boolean)
              .join(" ");

            const fileLabel = vRow === 7 ? FILES[c] : "";
            const rankLabel = vCol === 0 ? String(r + 1) : "";

            return (
              <div
                key={index}
                className={squareClass}
                onClick={() => onSquareClick?.(index)}
              >
                {fileLabel && (
                  <span
                    className={`${styles.coordLabel} ${styles.fileLabel} ${
                      isDark ? styles.coordDark : styles.coordLight
                    }`}
                  >
                    {fileLabel}
                  </span>
                )}
                {rankLabel && (
                  <span
                    className={`${styles.coordLabel} ${styles.rankLabel} ${
                      isDark ? styles.coordDark : styles.coordLight
                    }`}
                  >
                    {rankLabel}
                  </span>
                )}
                {isValidTarget &&
                  (piece ? (
                    <div className={styles.captureHint} />
                  ) : (
                    <div className={styles.moveHint} />
                  ))}
                {piece && (
                  <Piece
                    key={isSliding ? `anim-${animateId}-${index}` : `piece-${index}`}
                    piece={piece}
                    sliding={isSliding}
                    slideOffset={slideOffset}
                  />
                )}
              </div>
            );
          })
        )}

        {animInfo && animInfo.capturedPiece && ghostPos && (
          <div
            key={`ghost-${animateId}`}
            className={styles.ghostLayer}
            style={{
              left: `${ghostPos.vCol * 12.5}%`,
              top: `${ghostPos.vRow * 12.5}%`,
            }}
          >
            <img
              className={styles.ghostPiece}
              src={pieceImageUrl(animInfo.capturedPiece)}
              alt=""
              draggable={false}
            />
          </div>
        )}
      </div>

      {arrow && <MoveArrow from={arrow.from} to={arrow.to} flipped={flipped} />}
    </div>
  );
};

const Piece = ({
  piece,
  sliding,
  slideOffset,
}: {
  piece: ChessPiece;
  sliding: boolean;
  slideOffset: { dx: string; dy: string } | null;
}) => {
  const style: React.CSSProperties = sliding && slideOffset
    ? {
        ["--slide-dx" as string]: slideOffset.dx,
        ["--slide-dy" as string]: slideOffset.dy,
      }
    : {};

  return (
    <img
      className={`${styles.piece} ${sliding ? styles.sliding : ""}`}
      style={style}
      src={pieceImageUrl(piece)}
      alt={`${piece.color} ${piece.type}`}
      draggable={false}
    />
  );
};

const MoveArrow = ({
  from,
  to,
  flipped,
}: {
  from: number;
  to: number;
  flipped: boolean;
}) => {
  const fromVC = visualRowCol(from, flipped);
  const toVC = visualRowCol(to, flipped);
  const x1 = (fromVC.vCol + 0.5) * 12.5;
  const y1 = (fromVC.vRow + 0.5) * 12.5;
  const x2 = (toVC.vCol + 0.5) * 12.5;
  const y2 = (toVC.vRow + 0.5) * 12.5;

  const dx = x2 - x1;
  const dy = y2 - y1;
  const len = Math.sqrt(dx * dx + dy * dy);
  const ux = dx / len;
  const uy = dy / len;
  const margin = 4;
  const sx = x1 + ux * margin;
  const sy = y1 + uy * margin;
  const ex = x2 - ux * (margin + 2);
  const ey = y2 - uy * (margin + 2);

  const arrowLen = 5;
  const a1x = ex - ux * arrowLen - uy * 3;
  const a1y = ey - uy * arrowLen + ux * 3;
  const a2x = ex - ux * arrowLen + uy * 3;
  const a2y = ey - uy * arrowLen - ux * 3;

  return (
    <svg className={styles.arrowOverlay} viewBox="0 0 100 100" preserveAspectRatio="none">
      <line x1={sx} y1={sy} x2={ex} y2={ey} className={styles.arrowLine} />
      <polygon
        points={`${ex},${ey} ${a1x},${a1y} ${a2x},${a2y}`}
        className={styles.arrowHead}
      />
    </svg>
  );
};