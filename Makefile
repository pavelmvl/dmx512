
dmx512: cmd/dmx512/dmx512.go
	go build -o $@ $<

dmx512.exe: cmd/dmx512/dmx512.go
	CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui" -o $@ $<
