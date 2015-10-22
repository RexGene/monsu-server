#!/bin/bash

pid=$(ps fx | grep -v "grep" |grep ./monsu-server | awk '{print $1}')
if [ "$pid" == "" ]; then
    exit 1
fi

kill "$pid"
exit 0
