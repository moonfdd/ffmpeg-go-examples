
.PHONY: all

all: cmd build

cmd build:
	go build -buildmode=exe -ldflags="-s -w" -tags="tempdll" -o govclvideo.exe ./examples/govclvideo/govclui/code