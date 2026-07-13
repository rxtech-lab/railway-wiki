#!/bin/zsh
set -euo pipefail

SCRIPT_DIR=${0:A:h}
IOS_DIR=${SCRIPT_DIR:h}
SOURCE_PACKAGES_DIR=${1:-${SOURCE_PACKAGES_DIR:-}}
RESOLVED=${IOS_DIR}/iOS.xcodeproj/project.xcworkspace/xcshareddata/swiftpm/Package.resolved

if [[ -z "${SOURCE_PACKAGES_DIR}" ]]; then
    print -u2 "usage: $0 <cloned-source-packages-directory>"
    exit 64
fi

EXPECTED_VERSION=$(python3 -c '
import json
import sys

with open(sys.argv[1], encoding="utf-8") as resolved:
    pins = json.load(resolved)["pins"]
for pin in pins:
    if pin["identity"] == "swiftlintplugins":
        print(pin["state"]["version"])
        break
else:
    raise SystemExit("SwiftLintPlugins is not pinned in Package.resolved")
' "${RESOLVED}")

SWIFTLINT=${SOURCE_PACKAGES_DIR}/artifacts/swiftlintplugins/SwiftLintBinary/SwiftLintBinary.artifactbundle/macos/swiftlint
if [[ ! -x "${SWIFTLINT}" ]]; then
    print -u2 "resolved SwiftLint artifact not found at ${SWIFTLINT}"
    exit 1
fi

ACTUAL_VERSION=$("${SWIFTLINT}" version)
if [[ "${ACTUAL_VERSION}" != "${EXPECTED_VERSION}" ]]; then
    print -u2 "SwiftLint artifact version ${ACTUAL_VERSION} does not match resolved version ${EXPECTED_VERSION}"
    exit 1
fi

cd "${IOS_DIR}"
exec "${SWIFTLINT}" lint --strict --no-cache --config .swiftlint.yml
