#!/usr/bin/env bash

# This script assumes the docker runtime binary in the PATH.

# Only build the docker image in case it doesn't exist
docker inspect --type=image xds-relay:latest > /dev/null
if [ $? -ne 0 ]; then
    make build-docker-image
fi

docker run -v $(pwd):/xds-relay -it xds-relay:latest $*
