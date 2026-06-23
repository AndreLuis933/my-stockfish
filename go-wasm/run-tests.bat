@echo off
setlocal enabledelayedexpansion

REM ============================================================
REM  Engine comparison tests — current vs v3 (reference)
REM  v3 is the baseline: new versions test against it.
REM  Usage:
REM    run-tests.bat                - tests engines\v4.exe vs v3
REM    run-tests.bat myengine.exe   - tests engines\myengine.exe vs v3
REM ============================================================

set "ENGINES_DIR=%~dp0engines"
set "RESULTS_DIR=%~dp0results"
set "OPENINGS=%ENGINES_DIR%\aberturas.epd"
set "REF_NAME=v3"

REM Pick the candidate engine: arg1 if provided, else v4.exe
if "%~1"=="" (
    set "CANDIDATE=%ENGINES_DIR%\v4.exe"
    set "CAND_NAME=v4"
) else (
    set "CANDIDATE=%ENGINES_DIR%\%~1"
    set "CAND_NAME=%~n1"
)

if not exist "%CANDIDATE%" (
    echo.
    echo ERROR: Candidate engine not found: %CANDIDATE%
    echo Place the new engine .exe in the engines\ folder and run:
    echo   run-tests.bat yourengine.exe
    echo   run-tests.bat              ^(defaults to v4.exe^)
    echo.
    goto :eof
)

if not exist "%RESULTS_DIR%" mkdir "%RESULTS_DIR%"

REM Timestamp for this run
for /f "usebackq" %%a in (`powershell -NoProfile -Command "Get-Date -Format yyyy-MM-dd_HHmm"`) do set "STAMP=%%a"

set "SUMMARY=%RESULTS_DIR%\summary.txt"
set "LOG=%RESULTS_DIR%\%CAND_NAME%_vs_%REF_NAME%-%STAMP%.log"
set "PGN=%RESULTS_DIR%\%CAND_NAME%_vs_%REF_NAME%-%STAMP%.pgn"

REM Write header to summary
echo ============================================ > "%SUMMARY%"
echo  Engine Comparison — %STAMP% >> "%SUMMARY%"
echo ============================================ >> "%SUMMARY%"
echo  Reference         : %REF_NAME% >> "%SUMMARY%"
echo  Candidate         : %CAND_NAME% >> "%SUMMARY%"
echo  Rounds per match  : 100 >> "%SUMMARY%"
echo  Time control      : tc=5+1 (5s per game + 1s increment) >> "%SUMMARY%"
echo  Concurrency       : 4 >> "%SUMMARY%"
echo  Openings          : aberturas.epd (10 positions, random order) >> "%SUMMARY%"
echo ============================================ >> "%SUMMARY%"
echo. >> "%SUMMARY%"

REM Common cutechess flags
set "COMMON=-each proto=uci tc=5+1 -rounds 100 -concurrency 4 -openings file=%OPENINGS% format=epd order=random -draw movenumber=30 movecount=8 score=20 -recover"

echo.
echo === Match: %CAND_NAME% vs %REF_NAME% ===
cutechess-cli -engine name=%CAND_NAME% cmd="%CANDIDATE%" -engine name=%REF_NAME% cmd="%ENGINES_DIR%\%REF_NAME%.exe" %COMMON% -pgnout "%PGN%" > "%LOG%" 2>&1
type "%LOG%"
call :parseScore "%LOG%" %CAND_NAME% %REF_NAME%
echo %CAND_NAME% vs %REF_NAME% : !SCORE_LINE! >> "%SUMMARY%"

echo. >> "%SUMMARY%"
echo PGN:  %PGN% >> "%SUMMARY%"
echo Log:  %LOG% >> "%SUMMARY%"
echo. >> "%SUMMARY%"
REM To compute ELO + LOS after installing bayeselo:
REM bayeselo %RESULTS_DIR%\*.pgn

echo.
echo ============================================
echo  Match complete. Summary:
echo ============================================
type "%SUMMARY%"
echo.
echo PGN:  %PGN%
echo Log:  %LOG%
echo.
endlocal
goto :eof

REM ============================================================
REM  Parse cutechess final score line from the log
REM  cutechess prints: "Score of engine1 vs engine2: W - L - D ..."
REM  %1 = log file, %2 = engine1 name, %3 = engine2 name
REM ============================================================
:parseScore
set "LOG=%~1"
set "SCORE_LINE=see log"
for /f "usebackq tokens=*" %%L in ("%LOG%") do (
    echo %%L | findstr /c:"Score of" >nul && (
        set "RAW=%%L"
        set "SCORE_LINE=!RAW!"
    )
)
goto :eof