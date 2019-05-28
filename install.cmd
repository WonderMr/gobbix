nssm64 install "gobbix" "%CD%\HaspForwarder.exe" 
nssm64 set "gobbix" AppDirectory "\"%CD%\"" 
sc start "gobbix"
pause