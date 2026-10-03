#!/bin/sh
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$root"
if grep -RInE --exclude-dir=node_modules --exclude-dir=.git --exclude-dir=dist \
  'AKIA[0-9A-Z]{16}|-----BEGIN (RSA |OPENSSH )?PRIVATE KEY-----|xox[baprs]-' .; then
  echo "secret-scan: possible secret found" >&2
  exit 1
fi
echo "secret-scan: no high-confidence secrets found"
