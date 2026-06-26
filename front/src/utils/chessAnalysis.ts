export type MoveQuality = "good" | "inaccuracy" | "mistake" | "blunder";

export const BLUNDER_THRESHOLD = 200;
export const MISTAKE_THRESHOLD = 100;
export const INACCURACY_THRESHOLD = 50;

export const classifyMove = (evalBefore: number, evalAfter: number): MoveQuality => {
  const swing = evalBefore - evalAfter;
  if (swing >= BLUNDER_THRESHOLD) return "blunder";
  if (swing >= MISTAKE_THRESHOLD) return "mistake";
  if (swing >= INACCURACY_THRESHOLD) return "inaccuracy";
  return "good";
};

export const moveQualitySymbol = (quality: MoveQuality): string => {
  switch (quality) {
    case "blunder":
      return "??";
    case "mistake":
      return "?";
    case "inaccuracy":
      return "?!";
    default:
      return "";
  }
};

export const formatEval = (score: number): string => {
  const pawns = score / 100;
  const sign = pawns >= 0 ? "+" : "";
  return `${sign}${pawns.toFixed(2)}`;
};