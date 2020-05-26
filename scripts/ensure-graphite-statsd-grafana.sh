#!/usr/bin/env bash

docker inspect -f '{{.State.Running}}' graphite > /dev/null
if [ $? -ne 0 ]; then
    docker run -d\
           --name graphite\
           --restart=always\
           -p 80:80\
           -p 81:81\
           -p 2003-2004:2003-2004\
           -p 2023-2024:2023-2024\
           -p 8125:8125/udp\
           -p 8126:8126\
           hopsoft/graphite-statsd
fi

echo "Open http://localhost:80 to open grafana or http://localhost:81 to open graphite.
For more details on how to use grafana/graphite, visit https://hub.docker.com/r/hopsoft/graphite-statsd/."
