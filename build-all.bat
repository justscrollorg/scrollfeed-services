@echo off
echo === Build Verification Script ===
echo.

echo Building news-service...
cd news-service
go build -o news-service.exe ./cmd
if %ERRORLEVEL% NEQ 0 (
    echo ❌ news-service build failed
    exit /b 1
) else (
    echo ✅ news-service built successfully
)

echo.
echo Building news-fetcher-service...
cd ..\news-fetcher-service
go build -o news-fetcher.exe ./cmd
if %ERRORLEVEL% NEQ 0 (
    echo ❌ news-fetcher-service build failed
    exit /b 1
) else (
    echo ✅ news-fetcher-service built successfully
)

echo.
echo Building video-service...
cd ..\video-service
go build -o video-service.exe ./cmd
if %ERRORLEVEL% NEQ 0 (
    echo ❌ video-service build failed
    exit /b 1
) else (
    echo ✅ video-service built successfully
)

cd ..
echo.
echo === All Services Built Successfully! ===
echo.
echo Ready for deployment:
echo - news-service/news-service.exe
echo - news-fetcher-service/news-fetcher.exe  
echo - video-service/video-service.exe
