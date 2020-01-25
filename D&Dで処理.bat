rem @echo off
"%~dp0\isbn2title.exe" -save %1
if %~n1 == 新しいフォルダー mkdir 新しいフォルダー
pause
