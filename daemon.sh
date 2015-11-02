#!/bin/bash

if [ $# -ne 2 ]; then
    echo "$0 <name> <sleep-time>"
    exit 1
fi  

while true
do
    result=$(ps -fe | grep "$1" | grep -v "grep" | grep -v "$0" | wc -l )
    if [ $result == 0 ]; then
        if [ -f "log" ]; then

          if [ ! -d "log_backup" ]; then
            mkdir log_backup
          fi

          time=$(date +%Y-%m-%d_%H-%M-%S)


          mv "log" "log_backup/$time.log"
        fi
    
        ./start.sh
    fi

    sleep $2
done

