@echo off

echo Building ss.exe (strip debug info)...
go build -ldflags="-s -w" -trimpath -o ss.exe ./cmd/ss
if %ERRORLEVEL% neq 0 exit /b %ERRORLEVEL%

where upx >nul 2>nul
if %ERRORLEVEL% equ 0 (
    echo Compressing ss.exe with UPX...
    upx --best -q ss.exe
    if %ERRORLEVEL% neq 0 (
        echo Warning: UPX compression failed, using uncompressed binary.
    )
) else (
    echo UPX not found in PATH, skipping compression.
)

echo Build complete.