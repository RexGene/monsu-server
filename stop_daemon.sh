#!/bin/bash

pid=$(ps fx | grep -v "grep" |grep ./daemon | awk '{print $1}')
if [ "$pid" == "" ]; then
    exit 1
fi

kill "$pid"


./stop.sh

exit 0
