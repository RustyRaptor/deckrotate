#!/usr/bin/env bash
# Runs the plasmoid's DBusBackend against the real Go service on a private bus.
set -euo pipefail
cd "$(dirname "$0")/.."

if [[ -z "${DECKROTATE_IN_PRIVATE_BUS:-}" ]]; then
    exec env DECKROTATE_IN_PRIVATE_BUS=1 dbus-run-session -- "$0" "$@"
fi

bin=$(mktemp -d)
trap 'kill "${svc:-}" 2>/dev/null || true; rm -rf "$bin"' EXIT
go build -o "$bin/testservice" ./internal/service/testservice

coproc SVC { exec "$bin/testservice"; }
svc=$SVC_PID
read -r -t 10 line <&"${SVC[0]}"
[[ "$line" == ready ]] || { echo "testservice did not start" >&2; exit 1; }

QT_QPA_PLATFORM=offscreen /usr/lib/qt6/bin/qmltestrunner -input plasmoid/itests 2>&1 | grep -v Fontconfig
exit "${PIPESTATUS[0]}"
