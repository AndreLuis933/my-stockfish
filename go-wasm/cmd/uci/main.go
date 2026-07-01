package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"webassemble/pkg/book"
	"webassemble/pkg/engine"
)

const (
	engineName   = "my-stockfish"
	engineAuthor  = "Andre"
	debugEnabled = true // write a per-engine debug log to engines-debug/
)

var debugLog *logFile

// logFile is a thread-safe debug log writer. When debug is disabled it is nil
// and all helpers are no-ops (nil-pointer guard in each helper).
type logFile struct {
	mu sync.Mutex
	f  *os.File
}

func (l *logFile) write(format string, args ...any) {
	if l == nil || l.f == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	ts := time.Now().Format("15:04:05.000")
	fmt.Fprintf(l.f, ts+" "+format+"\n", args...)
}

func dbg(format string, args ...any) { debugLog.write(format, args...) }

// uciSession holds the state of a UCI conversation: the current position
// and the search goroutine lifecycle. The position is owned by the main
// goroutine (reading stdin); the search runs in its own goroutine and
// communicates only via channels.
type uciSession struct {
	pos  engine.Position
	tt   *engine.TranspositionTable
	book *book.Book
	rng  *rand.Rand

	searchMu   sync.Mutex
	stopCh     chan struct{}
	searchDone chan struct{}
}

func newSession(bookPath string) *uciSession {
	s := &uciSession{
		tt:  engine.DefaultTranspositionTable(),
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	s.pos.LoadFen(engine.StartingFEN)

	if bookPath != "" {
		b, err := book.LoadFile(bookPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not load opening book %s: %v\n", bookPath, err)
		} else {
			s.book = b
			dbg("loaded opening book: %s (%d entries)", bookPath, b.Len())
		}
	}

	return s
}

func initDebug() {
	if !debugEnabled {
		return
	}
	dir := "engines-debug"
	if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
		fmt.Fprintln(os.Stderr, "debug log mkdir:", mkErr)
		return
	}
	exe, _ := os.Executable()
	name := strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
	ts := time.Now().Format("2006-01-02_150405")
	pid := os.Getpid()
	path := filepath.Join(dir, name+"-"+ts+"-"+fmt.Sprintf("%d", pid)+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "debug log open:", err)
		return
	}
	debugLog = &logFile{f: f}
	dbg("=== engine start: %s pid=%d ts=%s ===", name, pid, ts)
}

func main() {
	bookPath := flag.String("book", "", "path to polyglot opening book .bin file")
	flag.Parse()

	initDebug()
	defer func() {
		if debugLog != nil && debugLog.f != nil {
			debugLog.f.Close()
		}
	}()

	s := newSession(*bookPath)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		raw := scanner.Text()
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		// Catch panics in command handling so the process doesn't die
		// silently and leave cutechess with a "disconnect" result.
		func() {
			defer func() {
				if r := recover(); r != nil {
					buf := make([]byte, 4096)
					n := runtime.Stack(buf, false)
					stack := string(buf[:n])
					dbg("PANIC in command %q: %v\n%s", line, r, stack)
					fmt.Fprintf(os.Stderr, "uci panic: %v\n%s\n", r, stack)
					fmt.Println("info string panic recovered")
					os.Stdout.Sync()
				}
			}()

			parts := strings.Fields(line)
			cmd := parts[0]
			args := parts[1:]

			switch cmd {
			case "uci":
				s.handleUci()
			case "isready":
				s.handleIsready()
			case "ucinewgame":
				s.handleUcinewgame()
			case "position":
				dbg("position: %s", line)
				s.handlePosition(args)
			case "go":
				dbg("go: %s | pos fen=%s", line, s.pos.FenSnapshot())
				s.handleGo(args)
			case "stop":
				dbg("stop")
				s.handleStop()
			case "quit":
				dbg("quit")
				s.stopSearch()
				return
			case "debug":
				// debug on/off — acknowledged, no-op.
			}
			// Unknown commands are ignored per UCI spec.
		}()
	}
	if err := scanner.Err(); err != nil {
		dbg("stdin scanner error: %v", err)
		fmt.Fprintln(os.Stderr, "stdin error:", err)
	}
	dbg("=== engine exit ===")
}