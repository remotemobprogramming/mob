@echo off
@setlocal

REM probably there is a better place
set target=%SystemRoot%

go build mob.go
copy mob.exe %target%
echo "installed 'mob' to %target%"