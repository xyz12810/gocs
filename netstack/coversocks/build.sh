#!/bin/bash
if [ "$1" = "android" ]; then
    gomobile bind -o coversocks.aar -target=android github.com/coversocks/gocs/netstack/coversocks
else
    gomobile bind -target=ios -ldflags="-s -w" github.com/coversocks/gocs/netstack/coversocks
    du -h Coversocks.framework
fi
