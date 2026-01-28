#!/usr/bin/env bash
# Build script that suppresses duplicate library warnings
CGO_LDFLAGS="-Wl,-w" go build "$@"
