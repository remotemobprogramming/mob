@echo off
@setlocal

echo Installing 'mob'...

set target="%USERPROFILE%\.mob"

if not exist %target% (
	md %target%
	echo Created %target%
)

go build mob.go
copy mob.exe %target%
echo 'mob.exe' installed to %target%

REM add target to PATH
for /f "tokens=*" %%a in ('echo "%PATH%" ^| find /C /I %target%') do set INPATH=%%a

setx MOB_HOME %target%
echo Added '%target%' to PATH


if %INPATH% EQU 0 (
	setx PATH %%MOB_HOME%%;"%PATH%"
	echo Added '%target%' to PATH
)
echo Installed 'mob' successfully
pause