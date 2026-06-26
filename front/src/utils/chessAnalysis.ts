export const formatEval = (score: number): string => {
  const pawns = score / 100;
  const sign = pawns >= 0 ? "+" : "";
  return `${sign}${pawns.toFixed(2)}`;
};