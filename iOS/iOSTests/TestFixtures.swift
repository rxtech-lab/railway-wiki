import Foundation
import XCTest
@testable import iOS

@MainActor
enum TestFixtures {
    static let configurationValues: [String: String] = [
        "RailwayAPIBaseURL": "https://api.example.test/root",
        "OverpassAPIBaseURL": "https://api.example.test/api/management/overpass",
        "OSMStackBaseURL": "https://maps.example.test",
        "OSMMapStylePath": "/maps/styles/basic/style.json",
        "OSMMapsAPIKey": "maps-only-key",
        "AuthIssuerURL": "https://auth.example.test",
        "AuthClientID": "railway-wiki-tests",
        "AuthRedirectURI": "railwaywiki://oauth-callback",
        "AuthScopes": "openid profile email"
    ]

    static func configuration(
        environment: [String: String] = ["E2E_AUTH_TOKEN": "test-admin-token"],
        launchOverrides: [String: String] = [:]
    ) throws -> AppConfiguration {
        try AppConfiguration(
            values: configurationValues,
            environment: environment,
            launchOverrides: launchOverrides
        )
    }

    static func record(
        id: String,
        name: String,
        additionalValues: [String: JSONValue] = [:]
    ) -> ResourceRecord {
        ResourceRecord(values: ["id": .string(id), "name": .string(name)].merging(additionalValues) { _, new in new })
    }
}

final class StubURLProtocol: URLProtocol, @unchecked Sendable {
    typealias Handler = (URLRequest) throws -> (HTTPURLResponse, Data)

    private static let lock = NSLock()
    nonisolated(unsafe) private static var requestHandler: Handler?

    static func install(_ handler: @escaping Handler) {
        lock.performLocked { requestHandler = handler }
    }

    static func reset() {
        lock.performLocked { requestHandler = nil }
    }

    static func session() -> URLSession {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.protocolClasses = [StubURLProtocol.self]
        return URLSession(configuration: configuration)
    }

    override class func canInit(with request: URLRequest) -> Bool { true }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }

    override func startLoading() {
        do {
            guard let handler = Self.lock.performLocked({ Self.requestHandler }) else {
                throw URLError(.resourceUnavailable)
            }
            let (response, data) = try handler(request)
            client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
            client?.urlProtocol(self, didLoad: data)
            client?.urlProtocolDidFinishLoading(self)
        } catch {
            client?.urlProtocol(self, didFailWithError: error)
        }
    }

    override func stopLoading() {}

    static func response(
        for request: URLRequest,
        status: Int = 200,
        json: String
    ) throws -> (HTTPURLResponse, Data) {
        let response = try XCTUnwrap(
            HTTPURLResponse(
                url: request.url ?? URL(string: "https://invalid.example")!,
                statusCode: status,
                httpVersion: "HTTP/1.1",
                headerFields: ["Content-Type": "application/json"]
            )
        )
        return (response, Data(json.utf8))
    }
}

private extension NSLock {
    func performLocked<Value>(_ operation: () throws -> Value) rethrows -> Value {
        lock()
        defer { unlock() }
        return try operation()
    }
}

extension URLRequest {
    func bodyData() throws -> Data {
        if let httpBody { return httpBody }
        guard let stream = httpBodyStream else { throw URLError(.cannotDecodeContentData) }
        stream.open()
        defer { stream.close() }
        var result = Data()
        var buffer = [UInt8](repeating: 0, count: 4_096)
        while true {
            let count = stream.read(&buffer, maxLength: buffer.count)
            if count < 0 { throw stream.streamError ?? URLError(.cannotDecodeContentData) }
            if count == 0 { return result }
            result.append(buffer, count: count)
        }
    }
}
