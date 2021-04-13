#wget "http://192.168.11.183:8000/api/v1/exitnow"
#del edosapi.exe X:\edosapi.exe 
go generate
go build -ldflags="-s -w" .
#copy edosapi.exe X:\