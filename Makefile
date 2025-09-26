.PHONY: server connector all

GO_BIN=/opt/go/bin/go


server:
	gnome-terminal -- bash -c "cd server && $(GO_BIN) run .; exec bash"

connector:
	gnome-terminal -- bash -c "cd connector && $(GO_BIN) run .; exec bash"

all: server connector
