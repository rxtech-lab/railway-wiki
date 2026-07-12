import Foundation
import Network

final class MockRailwayServer: @unchecked Sendable {
    private let listener: NWListener
    private let queue = DispatchQueue(label: "app.rxlab.railwaywiki.ui-fixture")
    private let lock = NSLock()
    private var connections: [NWConnection] = []
    private(set) var port: UInt16 = 0

    init() throws {
        listener = try NWListener(using: .tcp, on: .any)
        let ready = DispatchSemaphore(value: 0)
        let state = LockedState()
        listener.stateUpdateHandler = { newState in
            switch newState {
            case .ready:
                ready.signal()
            case .failed(let error):
                state.set(error: error)
                ready.signal()
            default:
                break
            }
        }
        listener.newConnectionHandler = { [weak self] connection in self?.accept(connection) }
        listener.start(queue: queue)
        guard ready.wait(timeout: .now() + 5) == .success else {
            listener.cancel()
            throw FixtureError.startTimedOut
        }
        if let error = state.error {
            listener.cancel()
            throw error
        }
        guard let boundPort = listener.port else {
            listener.cancel()
            throw FixtureError.missingPort
        }
        port = boundPort.rawValue
    }

    var baseURL: String { "http://127.0.0.1:\(port)" }

    func stop() {
        listener.cancel()
        lock.performLocked {
            connections.forEach { $0.cancel() }
            connections.removeAll()
        }
    }

    private func accept(_ connection: NWConnection) {
        lock.performLocked { connections.append(connection) }
        connection.start(queue: queue)
        receiveRequest(on: connection, accumulated: Data())
    }

    private func receiveRequest(on connection: NWConnection, accumulated: Data) {
        connection.receive(minimumIncompleteLength: 1, maximumLength: 65_536) { [weak self] data, _, isComplete, error in
            guard let self else { return }
            var requestData = accumulated
            if let data { requestData.append(data) }
            if self.hasCompleteHeaders(requestData) || isComplete || error != nil {
                self.respond(to: requestData, on: connection)
            } else {
                self.receiveRequest(on: connection, accumulated: requestData)
            }
        }
    }

    private func hasCompleteHeaders(_ data: Data) -> Bool {
        data.range(of: Data("\r\n\r\n".utf8)) != nil
    }

    private func respond(to requestData: Data, on connection: NWConnection) {
        let requestLine = (String(bytes: requestData, encoding: .utf8) ?? "")
            .components(separatedBy: "\r\n")
            .first ?? "GET / HTTP/1.1"
        let components = requestLine.split(separator: " ")
        let method = components.first.map(String.init) ?? "GET"
        let rawPath = components.count > 1 ? String(components[1]) : "/"
        let path = rawPath.split(separator: "?", maxSplits: 1).first.map(String.init) ?? rawPath
        let fixture = response(method: method, path: path)
        let body = Data(fixture.body.utf8)
        let headers = """
        HTTP/1.1 \(fixture.status) \(fixture.status == 200 ? "OK" : "Not Found")\r
        Content-Type: application/json\r
        Content-Length: \(body.count)\r
        Connection: close\r
        \r

        """
        var payload = Data(headers.utf8)
        payload.append(body)
        connection.send(content: payload, completion: .contentProcessed { [weak self] _ in
            connection.cancel()
            self?.lock.performLocked { self?.connections.removeAll { $0 === connection } }
        })
    }

    private func response(method: String, path: String) -> (status: Int, body: String) {
        if path == "/api/management/dashboard" { return (200, Self.dashboard) }
        if path == "/api/management/stations" && method == "GET" { return (200, Self.stations) }
        if path == "/api/management/stations/schema" { return (200, Self.stationSchema) }
        if path == "/api/management/stations" && method == "POST" { return (200, Self.createdStation) }
        if path == "/api/management/stations/station-central" { return (200, Self.centralStation) }
        if path.hasPrefix("/api/management/station-codes") ||
            path.hasPrefix("/api/management/platforms") ||
            path.hasPrefix("/api/management/route-stations") {
            return (200, Self.emptyPage)
        }
        if path == "/api/management/overpass/stations/node/123/import" && method == "POST" {
            return (200, Self.createdStation)
        }
        if path == "/api/management/overpass/stations" { return (200, Self.overpassCandidates) }
        if path == "/maps/styles/basic/style.json" { return (200, Self.mapStyle) }
        if path.hasPrefix("/api/management/") { return (200, Self.emptyPage) }
        return (404, #"{"error":"fixture route not found","code":"not_found"}"#)
    }

    private static let dashboard = #"""
    {
      "resources":[
        {"resource":"stations","count":2},
        {"resource":"routes","count":1}
      ],
      "stationCoverage":{"total":2,"withCoordinates":2},
      "generatedAt":"2026-07-12T00:00:00Z"
    }
    """#
    private static let stations = #"""
    {
      "items":[
        {
          "id":"station-central",
          "name":"Central",
          "nameEn":"Central Station",
          "stationNumber":"CEN",
          "latitude":22.2819,
          "longitude":114.1582
        },
        {"id":"station-admiralty","name":"Admiralty","latitude":22.2795,"longitude":114.1653}
      ],
      "pagination":{"next":null}
    }
    """#
    private static let centralStation = #"""
    {
      "id":"station-central",
      "name":"Central",
      "nameEn":"Central Station",
      "stationNumber":"CEN",
      "latitude":22.2819,
      "longitude":114.1582
    }
    """#
    private static let createdStation = #"""
    {"id":"station-created","name":"Fixture Station","latitude":22.30,"longitude":114.17}
    """#
    private static let emptyPage = #"{"items":[],"pagination":{"next":null}}"#
    private static let stationSchema = #"""
    {
      "type":"object",
      "required":["name","latitude","longitude"],
      "properties":{
        "name":{"type":"string","title":"Name"},
        "nameEn":{"type":"string","title":"English Name"},
        "stationNumber":{"type":"string","title":"Station Number"},
        "latitude":{"type":"number","title":"Latitude"},
        "longitude":{"type":"number","title":"Longitude"}
      },
      "x-ui-schema":{"ui:order":["name","nameEn","stationNumber","latitude","longitude"]}
    }
    """#
    private static let overpassCandidates = #"""
    {
      "items":[{
        "elementType":"node",
        "elementId":123,
        "name":"Fixture Halt",
        "railway":"halt",
        "latitude":22.29,
        "longitude":114.16,
        "tags":{"railway":"halt"}
      }],
      "generatedAt":"2026-07-12T00:00:00Z"
    }
    """#
    private static let mapStyle = #"""
    {
      "version":8,
      "name":"Fixture",
      "sources":{},
      "layers":[{
        "id":"background",
        "type":"background",
        "paint":{"background-color":"#e8edf2"}
      }]
    }
    """#
}

private final class LockedState: @unchecked Sendable {
    private let lock = NSLock()
    private var storedError: Error?
    var error: Error? { lock.performLocked { storedError } }
    func set(error: Error) { lock.performLocked { storedError = error } }
}

private enum FixtureError: Error {
    case startTimedOut
    case missingPort
}

private extension NSLock {
    func performLocked<Value>(_ operation: () throws -> Value) rethrows -> Value {
        lock()
        defer { unlock() }
        return try operation()
    }
}
