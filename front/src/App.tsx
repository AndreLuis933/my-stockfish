import { useGame } from "@/hooks/useGame";
import { Board } from "@/components/Board/Board";
import styles from "./App.module.css";

function App() {
  const { state, handleSquareClick } = useGame();
  const { board, currentPlayer, selectedSquare, movesForSelected } = state;

  return (
    <div className={styles.page}>
      <div className={styles.turnBanner}>
        <div className={`${styles.dot} ${styles.dotWhite} ${currentPlayer === "white" ? styles.active : ""}`} />
        <span className={styles.turnText}>
          {currentPlayer === "white" ? "Vez das Brancas" : "Vez das Pretas"}
        </span>
        <div className={`${styles.dot} ${styles.dotBlack} ${currentPlayer === "black" ? styles.active : ""}`} />
      </div>

      <Board
        board={board}
        selectedSquare={selectedSquare}
        validMoveSquares={movesForSelected.map((m) => m.to)}
        onSquareClick={handleSquareClick}
      />
    </div>
  );
}

export default App;
