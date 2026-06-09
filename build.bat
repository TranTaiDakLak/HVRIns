@echo off
REM Build script — inject version theo format vMM.DD.HH.mm

for /f "tokens=1-5 delims=:./- " %%a in ("%date% %time%") do (
    set MM=%%b
    set DD=%%c
    set HH=%%d
    set NN=%%e
)

REM Pad HH với 0 nếu là số 1 chữ số
if "%HH:~0,1%"==" " set HH=0%HH:~1,1%

set VERSION=v%MM%.%DD%.%HH%.%NN%
echo Building version: %VERSION%

wails build -ldflags "-X main.AppVersion=%VERSION%"
