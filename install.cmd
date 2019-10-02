@echo off
@setlocal

echo Installing 'mob' ...

setx MOB_HOME "%USERPROFILE%\.mob"
set target="%MOB_HOME%"

if not exist %target% (
	md %target%
	echo Directory %target% created.
)

go build mob.go
copy mob.exe %target%
echo 'mob.exe' installed to %target%

REM add MOB_HOME to PATH
echo %path%|find /i "%MOB_HOME%">nul  || set path=%path%;%MOB_HOME%

echo 'mob' successfully installed.
pause