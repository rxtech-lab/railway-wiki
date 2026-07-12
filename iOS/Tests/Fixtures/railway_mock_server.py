#!/usr/bin/env python3
"""Deterministic HTTP fixture for Railway Wiki Maestro flows."""

import argparse
import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlparse


STATIONS = [
    {
        "id": "station-central",
        "name": "Central",
        "nameEn": "Central Station",
        "stationNumber": "CEN",
        "latitude": 22.2819,
        "longitude": 114.1582,
    },
    {
        "id": "station-admiralty",
        "name": "Admiralty",
        "latitude": 22.2795,
        "longitude": 114.1653,
    },
]

ROUTE = {"id": "route-harbour", "name": "Harbour Line", "routeType": "rail"}

STATION_SCHEMA = {
    "type": "object",
    "required": ["name", "latitude", "longitude"],
    "properties": {
        "name": {"type": "string", "title": "Name"},
        "nameEn": {"type": "string", "title": "English Name"},
        "stationNumber": {"type": "string", "title": "Station Number"},
        "latitude": {"type": "number", "title": "Latitude"},
        "longitude": {"type": "number", "title": "Longitude"},
    },
    "x-ui-schema": {
        "ui:order": ["name", "nameEn", "stationNumber", "latitude", "longitude"]
    },
}


class FixtureHandler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def do_GET(self):  # noqa: N802
        path = urlparse(self.path).path
        if path == "/healthz":
            return self.reply({"status": "ok"})
        if path == "/api/management/dashboard":
            return self.reply(
                {
                    "resources": [
                        {"resource": "stations", "count": len(STATIONS)},
                        {"resource": "routes", "count": 1},
                    ],
                    "stationCoverage": {"total": len(STATIONS), "withCoordinates": len(STATIONS)},
                    "generatedAt": "2026-07-12T00:00:00Z",
                }
            )
        if path == "/api/management/stations":
            return self.page(STATIONS)
        if path == "/api/management/stations/station-central":
            return self.reply(STATIONS[0])
        if path == "/api/management/stations/schema":
            return self.reply(STATION_SCHEMA)
        if path == "/api/management/routes":
            return self.page([ROUTE])
        if path == "/api/management/routes/route-harbour":
            return self.reply(ROUTE)
        if path == "/api/management/routes/route-harbour/configuration":
            return self.reply(
                {
                    "route": ROUTE,
                    "stations": [
                        {
                            "id": "route-station-central",
                            "routeId": "route-harbour",
                            "stationId": "station-central",
                            "sequence": 1,
                            "distanceFromStartKm": 0,
                        }
                    ],
                }
            )
        if path == "/api/management/overpass/stations":
            return self.reply(
                {
                    "items": [
                        {
                            "elementType": "node",
                            "elementId": 123,
                            "name": "Fixture Halt",
                            "railway": "halt",
                            "latitude": 22.29,
                            "longitude": 114.16,
                            "tags": {"railway": "halt"},
                        }
                    ],
                    "generatedAt": "2026-07-12T00:00:00Z",
                }
            )
        if path == "/maps/styles/basic/style.json":
            return self.reply(
                {
                    "version": 8,
                    "name": "Fixture",
                    "sources": {},
                    "layers": [
                        {
                            "id": "background",
                            "type": "background",
                            "paint": {"background-color": "#e8edf2"},
                        }
                    ],
                }
            )
        if path.startswith(
            (
                "/api/management/station-codes",
                "/api/management/platforms",
                "/api/management/route-stations",
            )
        ):
            return self.page([])
        if path.startswith("/api/management/"):
            return self.page([])
        return self.reply({"error": "fixture route not found", "code": "not_found"}, 404)

    def do_POST(self):  # noqa: N802
        path = urlparse(self.path).path
        self.read_body()
        if path.endswith("/configuration/validate"):
            return self.reply({"valid": True, "errors": []})
        if path.endswith("/node/123/import"):
            return self.reply(
                {
                    "id": "station-imported",
                    "name": "Fixture Halt",
                    "latitude": 22.29,
                    "longitude": 114.16,
                    "osmElementType": "node",
                    "osmElementId": 123,
                }
            )
        if path == "/api/management/stations":
            return self.reply(
                {"id": "station-created", "name": "Fixture Station", "latitude": 22.3, "longitude": 114.17},
                201,
            )
        return self.reply({"error": "fixture route not found", "code": "not_found"}, 404)

    def do_PUT(self):  # noqa: N802
        path = urlparse(self.path).path
        body = self.read_body()
        if path.endswith("/configuration"):
            return self.reply(body)
        if path.startswith("/api/management/"):
            return self.reply(body)
        return self.reply({"error": "fixture route not found", "code": "not_found"}, 404)

    def do_DELETE(self):  # noqa: N802
        self.read_body()
        self.reply(None, 204)

    def page(self, items):
        self.reply({"items": items, "pagination": {"next": None}})

    def read_body(self):
        length = int(self.headers.get("Content-Length", "0"))
        if length == 0:
            return {}
        return json.loads(self.rfile.read(length))

    def reply(self, value, status=200):
        payload = b"" if status == 204 else json.dumps(value, separators=(",", ":")).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        if payload:
            self.wfile.write(payload)

    def log_message(self, _format, *_args):
        return


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=18080)
    args = parser.parse_args()
    server = ThreadingHTTPServer((args.host, args.port), FixtureHandler)
    print(f"Railway Wiki fixture listening at http://{args.host}:{args.port}", flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()


if __name__ == "__main__":
    main()
