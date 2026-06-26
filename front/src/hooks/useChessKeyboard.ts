import { useEffect } from "react";

interface UseChessKeyboardParams {
  currentPly: number;
  historyLength: number;
  jumpToPly: (ply: number) => void;
  pendingPromotion: unknown;
}

export const useChessKeyboard = ({
  currentPly,
  historyLength,
  jumpToPly,
  pendingPromotion,
}: UseChessKeyboardParams): void => {
  useEffect(() => {
    if (pendingPromotion) return;

    const handler = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement) return;

      switch (e.key) {
        case "ArrowLeft":
          e.preventDefault();
          jumpToPly(currentPly - 1);
          break;
        case "ArrowRight":
          e.preventDefault();
          jumpToPly(currentPly + 1);
          break;
        case "Home":
          e.preventDefault();
          jumpToPly(0);
          break;
        case "End":
          e.preventDefault();
          jumpToPly(historyLength);
          break;
      }
    };

    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [currentPly, historyLength, jumpToPly, pendingPromotion]);
};