@echo off
@setlocal

echo Installing 'mob' ...

REM set variable for local script
set MOB_HOME=%USERPROFILE%\.mob
REM set for user, not visible for current shell
setx MOB_HOME "%MOB_HOME%"
set target="%MOB_HOME%"

if not exist %target% (
	md %target%
	echo Directory %target% created.
)

go build mob.go
copy mob.exe %target%
echo 'mob.exe' installed to %target%

REM add MOB_HOME to PATH, not used in current shell
echo %path%|find /i "%MOB_HOME%">nul || setx path "%path%;%MOB_HOME%"

echo 'mob' successfully installed.
pause