#!/usr/bin/env bash

CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

if [[ "$TMUX_WORMHOLE_DO_INSTALL" = "1" ]] ; then
    cd "$CURRENT_DIR"

    echo Trying to compile tmux-wormhole...
    echo
    GO111MODULE=on go build -o ./tmux-wormhole cmd/tmux-wormhole/main.go
    RES=$?

    echo
    if [[ -e ./tmux-wormhole ]] ; then
        echo Installed.
	read
        exit 0
    else
        echo Could not build tmux-wormhole.
	read
        exit 1
    fi
fi


DEFAULT_WORMHOLE_KEY=w

WORMHOLE_KEY="$(tmux show-option -gqv @wormhole-key)"
WORMHOLE_KEY=${WORMHOLE_KEY:-$DEFAULT_WORMHOLE_KEY}

tmux bind-key "${WORMHOLE_KEY}" run-shell -b "${CURRENT_DIR}/tmux-wormhole.sh"

if [[ ! -e "${CURRENT_DIR}/tmux-wormhole" ]] ; then
    tmux split-window "TMUX_WORMHOLE_DO_INSTALL=1 ${CURRENT_DIR}/tmux-wormhole.tmux"
fi
