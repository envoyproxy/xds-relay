#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

docker run -v $(pwd):/xds-relay -it xds-relay:latest $*
