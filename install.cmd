@echo off
@setlocal

REM probably there is a better place
set target=%SystemRoot%

go get github.com/blang/thymer
go build mob.go thymer.go
copy mob.exe %target%
echo "installed 'mob' to %target%"