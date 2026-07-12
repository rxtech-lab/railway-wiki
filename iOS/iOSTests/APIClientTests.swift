import Foundation
import XCTest
@testable import iOS

private let dashboardFixtureJSON = #"""
{
  "resources":[{"resource":"stations","count":2}],
  "stationCoverage":{"total":2,"withCoordinates":1},
  "generatedAt":"2026-07-12T00:00:00Z"
}
"""#

@MainActor
final class APIClientTests: XCTestCase {
    override func tearDown() {
        StubURLProtocol.reset()
        super.tearDown()
    }

    func testDashboardAddsBearerTokenAndDecodesSnapshot() async throws {
        StubURLProtocol.install { request in
            XCTAssertEqual(request.httpMethod, "GET")
            XCTAssertEqual(request.url?.path, "/root/api/management/dashboard")
            XCTAssertEqual(request.value(forHTTPHeaderField: "Authorization"), "Bearer test-admin-token")
            return try StubURLProtocol.response(for: request, json: dashboardFixtureJSON)
        }
        let client = try makeClient()

        let snapshot = try await client.dashboard()

        XCTAssertEqual(snapshot.stationCoverage.total, 2)
        XCTAssertEqual(snapshot.stationCoverage.withCoordinates, 1)
        XCTAssertEqual(snapshot.resources.first?.resource, "stations")
    }

    func testListSendsSearchBoundsCursorAndStableFilterOrder() async throws {
        StubURLProtocol.install { request in
            let components = try XCTUnwrap(URLComponents(url: try XCTUnwrap(request.url), resolvingAgainstBaseURL: false))
            XCTAssertEqual(request.url?.path, "/root/api/management/stations")
            XCTAssertEqual(components.queryItems?.map(\.name), [
                "limit", "q", "cursor", "south", "west", "north", "east", "companyId", "routeId"
            ])
            XCTAssertEqual(components.queryItems?.first(where: { $0.name == "q" })?.value, "central")
            return try StubURLProtocol.response(
                for: request,
                json: #"{"items":[],"pagination":{"next":null}}"#
            )
        }
        let client = try makeClient()
        let stations = try XCTUnwrap(ResourceDefinition.all.first { $0.id == "stations" })

        let page = try await client.list(
            stations,
            query: "central",
            cursor: "next-page",
            bounds: MapBounds(south: 22.2, west: 114.1, north: 22.4, east: 114.3),
            filters: ["routeId": "route-1", "companyId": "company-1"]
        )

        XCTAssertTrue(page.items.isEmpty)
        XCTAssertNil(page.pagination.next)
    }

    func testListOmitsSearchForResourceWithoutBackendSearch() async throws {
        StubURLProtocol.install { request in
            let components = try XCTUnwrap(URLComponents(url: try XCTUnwrap(request.url), resolvingAgainstBaseURL: false))
            XCTAssertEqual(request.url?.path, "/root/api/management/station-codes")
            XCTAssertNil(components.queryItems?.first { $0.name == "q" })
            return try StubURLProtocol.response(
                for: request,
                json: #"{"items":[],"pagination":{"next":null}}"#
            )
        }
        let client = try makeClient()
        let stationCodes = try XCTUnwrap(ResourceDefinition.all.first { $0.id == "station-codes" })

        _ = try await client.list(stationCodes, query: "ignored")
    }

    func testSingleTokenRefreshRetryAfterUnauthorized() async throws {
        let lock = NSLock()
        var requestCount = 0
        StubURLProtocol.install { request in
            let count = lock.performLocked {
                requestCount += 1
                return requestCount
            }
            if count == 1 {
                return try StubURLProtocol.response(
                    for: request,
                    status: 401,
                    json: #"{"error":"expired","code":"unauthorized"}"#
                )
            }
            return try StubURLProtocol.response(for: request, json: dashboardFixtureJSON)
        }
        let client = try makeClient()

        _ = try await client.dashboard()

        XCTAssertEqual(lock.performLocked { requestCount }, 2)
    }

    func testForbiddenIsSurfacedAsLossOfAdminAccess() async throws {
        StubURLProtocol.install { request in
            try StubURLProtocol.response(
                for: request,
                status: 403,
                json: #"{"error":"admin role required","code":"forbidden"}"#
            )
        }
        let configuration = try TestFixtures.configuration()
        let auth = AuthSession(configuration: configuration)
        let client = APIClient(
            configuration: configuration,
            auth: auth,
            session: StubURLProtocol.session()
        )

        do {
            _ = try await client.dashboard()
            XCTFail("Expected forbidden error")
        } catch {
            XCTAssertEqual(error as? APIError, .forbidden)
        }
        XCTAssertTrue(auth.hasLostAdminAccess)
    }

    func testAtomicRouteBodyUsesArrayOrderAndEditableRouteFields() async throws {
        StubURLProtocol.install { request in
            XCTAssertEqual(request.httpMethod, "POST")
            XCTAssertEqual(request.url?.path, "/root/api/management/routes/route-1/configuration/validate")
            let body = try request.bodyData()
            let object = try XCTUnwrap(JSONSerialization.jsonObject(with: body) as? [String: Any])
            let route = try XCTUnwrap(object["route"] as? [String: Any])
            let stations = try XCTUnwrap(object["stations"] as? [[String: Any]])
            XCTAssertNil(route["id"])
            XCTAssertEqual(route["name"] as? String, "Harbour Line")
            XCTAssertEqual(stations.compactMap { $0["stationId"] as? String }, ["station-b", "station-a"])
            XCTAssertEqual(stations.first?["distanceFromStartKm"] as? Double, 0)
            return try StubURLProtocol.response(for: request, json: #"{"valid":true,"errors":[]}"#)
        }
        let client = try makeClient()
        let configuration = RouteConfiguration(
            route: TestFixtures.record(id: "route-1", name: "Harbour Line"),
            stations: [
                .init(id: "link-b", routeId: "route-1", stationId: "station-b", sequence: 20, distanceFromStartKm: 0),
                .init(id: "link-a", routeId: "route-1", stationId: "station-a", sequence: 10, distanceFromStartKm: 4.5)
            ]
        )

        let result = try await client.validateRoute(id: "route-1", configuration: configuration)

        XCTAssertTrue(result.valid)
    }

    func testMediaUploadPresignsThenUploadsWithoutBearerCredential() async throws {
        let lock = NSLock()
        var requestCount = 0
        StubURLProtocol.install { request in
            let count = lock.performLocked {
                requestCount += 1
                return requestCount
            }
            if count == 1 {
                XCTAssertEqual(request.url?.path, "/root/api/management/media/upload-url")
                XCTAssertEqual(request.value(forHTTPHeaderField: "Authorization"), "Bearer test-admin-token")
                let target = #"""
                {
                  "uploadUrl":"https://storage.example.test/object",
                  "method":"PUT",
                  "headers":{"Content-Type":"image/jpeg"},
                  "url":"https://cdn.example.test/object",
                  "expiresAt":"2026-07-12T01:00:00Z"
                }
                """#
                return try StubURLProtocol.response(
                    for: request,
                    status: 201,
                    json: target
                )
            }
            XCTAssertEqual(request.url?.host, "storage.example.test")
            XCTAssertEqual(request.httpMethod, "PUT")
            XCTAssertNil(request.value(forHTTPHeaderField: "Authorization"))
            XCTAssertEqual(request.value(forHTTPHeaderField: "Content-Type"), "image/jpeg")
            return try StubURLProtocol.response(for: request, json: "{}")
        }
        let fileURL = FileManager.default.temporaryDirectory.appending(path: UUID().uuidString + ".jpg")
        try Data("fixture".utf8).write(to: fileURL)
        defer { try? FileManager.default.removeItem(at: fileURL) }

        let publicURL = try await makeClient().uploadMedia(
            fileURL: fileURL,
            contentType: "image/jpeg",
            mediaType: "image"
        )

        XCTAssertEqual(publicURL.absoluteString, "https://cdn.example.test/object")
        XCTAssertEqual(lock.performLocked { requestCount }, 2)
    }

    private func makeClient() throws -> APIClient {
        let configuration = try TestFixtures.configuration()
        return APIClient(
            configuration: configuration,
            auth: AuthSession(configuration: configuration),
            session: StubURLProtocol.session()
        )
    }
}

private extension NSLock {
    func performLocked<Value>(_ operation: () throws -> Value) rethrows -> Value {
        lock()
        defer { unlock() }
        return try operation()
    }
}
