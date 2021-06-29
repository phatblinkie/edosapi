go generate
go build -ldflags="-s -w" .
copy edosapi.exe edosapi_installer\bin\
copy edosapi.exe ..\installwixtest\edosapi_bin\