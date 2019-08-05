#!/bin/bash

if [ $(go env GOOS) == "darwin" ]; then
    docker-machine start
    eval $(docker-machine env)
fi

exec "$@"
