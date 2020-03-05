#!/usr/bin/env bash
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"
modified=$(git status --porcelain pkg)
if [[ -n "${modified}" ]]; then
    echo -e "\nerror: commit changes to compiled protobufs from ./scripts/generate-api-protos.sh"
    exit 1
fi
