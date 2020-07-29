#!/bin/bash
if [ "$1" = "android" ];then
    gomobile bind -o coversocks.aar -target=android github.com/coversocks/gocs/netstack/ios
else
    gomobile bind -target=ios github.com/coversocks/gocs/netstack/ios
fi
