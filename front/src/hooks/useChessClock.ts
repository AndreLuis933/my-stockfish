import { useCallback, useEffect, useRef, useState } from "react";
import type { ChessColor } from "@/types/chess";

export interface ClockConfig {
  enabled: boolean;
  initialMs: number;
  incrementMs: number;
}

export interface ClockState {
  white: number;
  black: number;
}

export interface UseChessClock {
  clocks: ClockState;
  flagFallen: ChessColor | null;
  start: () => void;
  stop: () => void;
  reset: (config: ClockConfig) => void;
  onMoveComplete: (color: ChessColor) => void;
  pauseForAiThinking: (paused: boolean) => void;
}

export const formatClock = (ms: number): string => {
  const clamped = Math.max(0, ms);
  const totalSeconds = Math.floor(clamped / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  const tenths = Math.floor((clamped % 1000) / 100);

  if (clamped < 10000) {
    return `${minutes}:${String(seconds).padStart(2, "0")}.${tenths}`;
  }
  return `${minutes}:${String(seconds).padStart(2, "0")}`;
};

const PRESET_TO_MS = (minutes: number): number => Math.round(minutes * 60_000);

export const minutesToMs = PRESET_TO_MS;

export const useChessClock = (
  config: ClockConfig,
  activeColor: ChessColor,
  gameStarted: boolean,
  gameOver: boolean,
): UseChessClock => {
  const [clocks, setClocks] = useState<ClockState>({
    white: config.initialMs,
    black: config.initialMs,
  });
  const [flagFallen, setFlagFallen] = useState<ChessColor | null>(null);

  const configRef = useRef(config);
  const runningRef = useRef(false);
  const lastTickRef = useRef<number | null>(null);
  const rafRef = useRef<number | null>(null);
  const pausedRef = useRef(false);
  const tickRef = useRef<() => void>(() => {});

  useEffect(() => {
    configRef.current = config;
  }, [config]);

  useEffect(() => {
    tickRef.current = () => {
      if (!runningRef.current || pausedRef.current) {
        lastTickRef.current = null;
        rafRef.current = requestAnimationFrame(tickRef.current);
        return;
      }

      const now = performance.now();
      if (lastTickRef.current === null) {
        lastTickRef.current = now;
        rafRef.current = requestAnimationFrame(tickRef.current);
        return;
      }

      const delta = now - lastTickRef.current;
      lastTickRef.current = now;

      setClocks((prev) => {
        const next = { ...prev };
        if (activeColor === "white") {
          next.white = prev.white - delta;
          if (next.white <= 0) {
            next.white = 0;
            runningRef.current = false;
            setFlagFallen("white");
          }
        } else {
          next.black = prev.black - delta;
          if (next.black <= 0) {
            next.black = 0;
            runningRef.current = false;
            setFlagFallen("black");
          }
        }
        return next;
      });

      rafRef.current = requestAnimationFrame(tickRef.current);
    };
  }, [activeColor]);

  useEffect(() => {
    if (!config.enabled || gameOver || flagFallen !== null) {
      runningRef.current = false;
      lastTickRef.current = null;
      return;
    }
    if (gameStarted) {
      runningRef.current = true;
    }
  }, [config.enabled, gameStarted, gameOver, flagFallen]);

  useEffect(() => {
    rafRef.current = requestAnimationFrame(tickRef.current);
    return () => {
      if (rafRef.current !== null) cancelAnimationFrame(rafRef.current);
    };
  }, []);

  const start = useCallback(() => {
    if (!configRef.current.enabled) return;
    runningRef.current = true;
    lastTickRef.current = null;
  }, []);

  const stop = useCallback(() => {
    runningRef.current = false;
    lastTickRef.current = null;
  }, []);

  const reset = useCallback((newConfig: ClockConfig) => {
    runningRef.current = false;
    lastTickRef.current = null;
    setFlagFallen(null);
    setClocks({ white: newConfig.initialMs, black: newConfig.initialMs });
  }, []);

  const onMoveComplete = useCallback(
    (color: ChessColor) => {
      if (!configRef.current.enabled) return;
      setClocks((prev) => {
        const inc = configRef.current.incrementMs;
        if (color === "white") {
          return { ...prev, white: prev.white + inc };
        }
        return { ...prev, black: prev.black + inc };
      });
      lastTickRef.current = null;
    },
    [],
  );

  const pauseForAiThinking = useCallback((paused: boolean) => {
    pausedRef.current = paused;
    if (paused) {
      lastTickRef.current = null;
    }
  }, []);

  return {
    clocks,
    flagFallen,
    start,
    stop,
    reset,
    onMoveComplete,
    pauseForAiThinking,
  };
};