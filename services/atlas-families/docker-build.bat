@ECHO OFF
FOR %%a IN ("%~dp0\.") DO SET "parent=%%~nxa"
IF "%1"=="NO-CACHE" (docker build --no-cache -f Dockerfile.dev --tag %parent%:latest .)
IF NOT "%1"=="NO-CACHE" (docker build -f Dockerfile.dev --tag %parent%:latest .)
