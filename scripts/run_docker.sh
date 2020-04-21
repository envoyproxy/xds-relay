#!/usr/bin/env bash

# This script assumes that the xds-relay:latest docker image exists.

set -o errexit
set -o nounset
set -o pipefail

docker run -v $(pwd):/xds-relay -it xds-relay:latest $*
