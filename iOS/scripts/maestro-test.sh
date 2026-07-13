#!/bin/zsh
set -euo pipefail

SCRIPT_DIR=${0:A:h}
IOS_DIR=${SCRIPT_DIR:h}
FIXTURE=${IOS_DIR}/Tests/Fixtures/railway_mock_server.py
FLOW_PATH=${1:-${IOS_DIR}/.maestro}
if [[ "${FLOW_PATH}" != /* ]]; then
    FLOW_PATH=${IOS_DIR}/${FLOW_PATH}
fi

python3 "${FIXTURE}" --port 18080 &
FIXTURE_PID=$!
trap 'kill ${FIXTURE_PID} 2>/dev/null || true' EXIT INT TERM

for attempt in {1..50}; do
    if curl --fail --silent http://127.0.0.1:18080/healthz >/dev/null; then
        break
    fi
    sleep 0.1
done

curl --fail --silent http://127.0.0.1:18080/healthz >/dev/null
if [[ -n "${MAESTRO_DEVICE_UDID:-}" ]]; then
    maestro --device "${MAESTRO_DEVICE_UDID}" test "${FLOW_PATH}"
else
    maestro test "${FLOW_PATH}"
fi
