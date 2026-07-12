#!/usr/bin/env python3
"""Print the UDID of the newest available simulator matching an exact name."""

import json
import re
import subprocess
import sys


def runtime_version(identifier: str) -> tuple[int, ...]:
    match = re.search(r"iOS-(\d+(?:-\d+)*)$", identifier)
    if not match:
        return ()
    return tuple(int(part) for part in match.group(1).split("-"))


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit(f"usage: {sys.argv[0]} <exact-device-name>")
    name = sys.argv[1]
    result = subprocess.run(
        ["xcrun", "simctl", "list", "devices", "available", "-j"],
        check=True,
        capture_output=True,
        text=True,
    )
    devices = json.loads(result.stdout)["devices"]
    candidates = [
        (runtime_version(runtime), device["udid"])
        for runtime, runtime_devices in devices.items()
        for device in runtime_devices
        if device.get("isAvailable") is not False and device.get("name") == name
    ]
    if not candidates:
        available = sorted(
            {
                device.get("name", "")
                for runtime_devices in devices.values()
                for device in runtime_devices
                if device.get("isAvailable") is not False
            }
        )
        raise SystemExit(f"simulator {name!r} not found; available devices: {', '.join(available)}")
    print(max(candidates)[1])


if __name__ == "__main__":
    main()
