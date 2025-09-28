.PHONY: server connector all

###### START control panel ########
GO_BIN=go
TERM ?= KDE  # set KDE or GNM
####### END control panel #########

ifeq ($(TERM),GNM)
  TERMINAL_SERV = gnome-terminal -- bash -c "cd server && $(GO_BIN) run .; exec bash"
  TERMINAL_CLI  = gnome-terminal -- bash -c "cd connector && $(GO_BIN) run .; exec bash"
else
  TERMINAL_SERV = konsole --hold -e bash -c "cd server && $(GO_BIN) run ." &
  TERMINAL_CLI  = konsole --hold -e bash -c "cd connector && $(GO_BIN) run ." &
endif

server:
	$(TERMINAL_SERV)

connector:
	$(TERMINAL_CLI)

all: server connector
