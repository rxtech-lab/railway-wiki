#!/usr/bin/env bash
#
# Runs the resource-switching XCUITest against the REAL railway-wiki server booted
# in E2E_MODE (local SQLite, management auth disabled, sample data seeded).
#
# XCUITest code runs inside the simulator and cannot spawn a host process, so this
# wrapper starts the Go server on the host first, waits for it to be healthy, then
# runs xcodebuild pointed at an iPad simulator (the bug only reproduces in the
# split-view / regular size class).
#
# Env overrides:
#   PORT                  server port (default 8080)
#   E2E_IOS_SIMULATOR     exact simulator name (default "iPad Pro 13-inch (M5)")
#   E2E_IOS_DESTINATION   full xcodebuild -destination string (bypasses lookup)
#   E2E_SERVER_LOG        path for the server log (default: a temp file)
#   E2E_XCODEBUILD_LOG    if set, tee xcodebuild output here (for CI artifacts)
#   E2E_RESULT_BUNDLE     if set, write the .xcresult bundle here (for CI artifacts)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IOS_DIR="$(dirname "$SCRIPT_DIR")"
SERVER_DIR="$(cd "$IOS_DIR/../server" && pwd)"
PORT="${PORT:-8080}"
DB="$(mktemp -u).db"
SERVER_LOG="${E2E_SERVER_LOG:-$(mktemp -t railway-e2e-ios-server).log}"

echo "==> Starting railway-wiki server in E2E_MODE on :$PORT"
echo "    db=$DB  log=$SERVER_LOG"
E2E_MODE=true DATABASE_URL="file:${DB}" PORT="$PORT" \
  go -C "$SERVER_DIR" run ./cmd/server/main.go >"$SERVER_LOG" 2>&1 &
SERVER_PID=$!

cleanup() {
  echo "==> Stopping server (pid $SERVER_PID)"
  kill "$SERVER_PID" 2>/dev/null || true
  wait "$SERVER_PID" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "==> Waiting for http://localhost:${PORT}/health"
for attempt in $(seq 1 60); do
  if curl --fail --silent "http://localhost:${PORT}/health" >/dev/null 2>&1; then
    echo "    server ready after ${attempt}s"
    break
  fi
  if ! kill -0 "$SERVER_PID" 2>/dev/null; then
    echo "!! server exited early; log:" >&2
    cat "$SERVER_LOG" >&2
    exit 1
  fi
  if [ "$attempt" -eq 60 ]; then
    echo "!! server did not become healthy; log:" >&2
    cat "$SERVER_LOG" >&2
    exit 1
  fi
  sleep 1
done

# Resolve an iPad simulator destination (split view is required for the bug).
if [ -n "${E2E_IOS_DESTINATION:-}" ]; then
  DESTINATION="$E2E_IOS_DESTINATION"
else
  SIM_NAME="${E2E_IOS_SIMULATOR:-iPad Pro 13-inch (M5)}"
  UDID="$(python3 "$SCRIPT_DIR/select-simulator.py" "$SIM_NAME")"
  DESTINATION="platform=iOS Simulator,id=$UDID"
  echo "==> Using simulator: $SIM_NAME ($UDID)"
fi

# Optional passthrough so CI can reuse its resolved-package / derived-data caches
# and collect artifacts.
EXTRA_ARGS=()
[ -n "${SOURCE_PACKAGES_DIR:-}" ] && EXTRA_ARGS+=(-clonedSourcePackagesDirPath "$SOURCE_PACKAGES_DIR")
[ -n "${DERIVED_DATA_DIR:-}" ] && EXTRA_ARGS+=(-derivedDataPath "$DERIVED_DATA_DIR")
if [ -n "${E2E_RESULT_BUNDLE:-}" ]; then
  rm -rf "$E2E_RESULT_BUNDLE"           # xcodebuild refuses to overwrite an existing bundle
  EXTRA_ARGS+=(-resultBundlePath "$E2E_RESULT_BUNDLE")
fi

echo "==> Running iOSUITests/ResourceSwitchingUITests"
run_xcodebuild() {
  xcodebuild test \
    -project "$IOS_DIR/iOS.xcodeproj" \
    -scheme iOS \
    -configuration Debug \
    -destination "$DESTINATION" \
    -only-testing:iOSUITests/ResourceSwitchingUITests \
    -parallel-testing-enabled NO \
    -skipPackagePluginValidation \
    ${EXTRA_ARGS[@]+"${EXTRA_ARGS[@]}"} \
    CODE_SIGN_IDENTITY="-" CODE_SIGNING_REQUIRED=NO
}

if [ -n "${E2E_XCODEBUILD_LOG:-}" ]; then
  set -o pipefail
  run_xcodebuild 2>&1 | tee "$E2E_XCODEBUILD_LOG"
else
  run_xcodebuild
fi
