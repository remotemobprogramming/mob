@echo off
@setlocal

echo Installing 'mob' ...

REM set target to the user's bin directory
set target="%USERPROFILE%\bin"

if not exist %target% (
    md %target%
    echo Directory %target% created.
)

go build mob.go
copy mob.exe %target%
echo 'mob.exe' installed to %target%

REM add the user's bin directory to PATH, not used in current shell
echo %path%|find /i "%USERPROFILE%\bin">nul || setx path "%path%;%USERPROFILE%\bin"

echo 'mob' successfully installed.
pause
