import type { ClockConfig } from "@/hooks/useChessClock";
import type { UseChessClock } from "@/hooks/useChessClock";
import { minutesToMs } from "@/hooks/useChessClock";
import styles from "./ChessShared.module.css";

const CLOCK_PRESETS = [
  { label: "Sem relógio", minutes: 0 },
  { label: "1 min", minutes: 1 },
  { label: "3 min", minutes: 3 },
  { label: "5 min", minutes: 5 },
  { label: "10 min", minutes: 10 },
  { label: "15 min", minutes: 15 },
];

const INCREMENT_PRESETS = [0, 2, 3, 5, 10];

interface ClockConfigPanelProps {
  clockMinutes: number;
  onClockMinutesChange: (minutes: number) => void;
  clockIncrement: number;
  onClockIncrementChange: (inc: number) => void;
  clockConfig: ClockConfig;
  clock: UseChessClock;
  gameStarted: boolean;
}

export const ClockConfigPanel = ({
  clockMinutes,
  onClockMinutesChange,
  clockIncrement,
  onClockIncrementChange,
  clockConfig,
  clock,
  gameStarted,
}: ClockConfigPanelProps) => {
  const handlePreset = (minutes: number) => {
    onClockMinutesChange(minutes);
    if (minutes > 0 && !gameStarted) {
      clock.reset({
        enabled: true,
        initialMs: minutesToMs(minutes),
        incrementMs: clockIncrement * 1000,
      });
    } else if (minutes === 0 && !gameStarted) {
      clock.reset({
        enabled: false,
        initialMs: 0,
        incrementMs: 0,
      });
    }
  };

  return (
    <div className={styles.clockConfig}>
      <span className={styles.configTitle}>Relógio</span>
      <div className={styles.presetRow}>
        {CLOCK_PRESETS.map((p) => (
          <button
            key={p.label}
            className={`${styles.presetButton} ${clockMinutes === p.minutes ? styles.presetButtonActive : ""}`}
            onClick={() => handlePreset(p.minutes)}
            disabled={gameStarted && clockConfig.enabled}
          >
            {p.label}
          </button>
        ))}
      </div>
      {clockConfig.enabled && (
        <div className={styles.presetRow}>
          <span className={styles.incrementLabel}>Incremento:</span>
          {INCREMENT_PRESETS.map((s) => (
            <button
              key={s}
              className={`${styles.presetButtonSmall} ${clockIncrement === s ? styles.presetButtonActive : ""}`}
              onClick={() => onClockIncrementChange(s)}
              disabled={gameStarted}
            >
              {s}s
            </button>
          ))}
        </div>
      )}
    </div>
  );
};