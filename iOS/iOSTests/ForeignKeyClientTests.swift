import Foundation
import JSONSchemaForm
import XCTest

@testable import iOS

@MainActor
final class ForeignKeyClientTests: XCTestCase {
    override func tearDown() {
        StubURLProtocol.reset()
        super.tearDown()
    }

    func testSearchMapsEndpointQueryCursorAndRecords() async throws {
        StubURLProtocol.install { request in
            let components = try XCTUnwrap(
                URLComponents(url: try XCTUnwrap(request.url), resolvingAgainstBaseURL: false))
            XCTAssertEqual(request.url?.path, "/root/api/management/stations")
            XCTAssertEqual(
                components.queryItems?.first(where: { $0.name == "q" })?.value, "tokyo")
            XCTAssertEqual(
                components.queryItems?.first(where: { $0.name == "cursor" })?.value, "page-2")
            return try StubURLProtocol.response(
                for: request,
                json: #"""
                    {"items":[
                        {"id":"s1","name":"Tokyo","nameEn":"Tokyo Station"},
                        {"name":"missing id, dropped"}
                    ],
                    "pagination":{"next":"page-3"}}
                    """#
            )
        }
        let client = ManagementForeignKeyClient(api: try makeAPIClient())

        let page = try await client.search(endpoint: "stations", query: "tokyo", cursor: "page-2")

        XCTAssertEqual(page.items.count, 1)
        XCTAssertEqual(page.items.first?.id, "s1")
        XCTAssertEqual(page.items.first?.title, "Tokyo")
        XCTAssertEqual(page.items.first?.subtitle, "Tokyo Station")
        XCTAssertEqual(page.nextCursor, "page-3")
        // The full record travels along for object-storing transforms.
        XCTAssertEqual(page.items.first?.raw?.object?["name"], .string("Tokyo"))
    }

    func testSearchUnknownEndpointReturnsEmptyPageWithoutRequest() async throws {
        StubURLProtocol.install { _ in
            XCTFail("no request expected for an unknown endpoint")
            throw URLError(.badURL)
        }
        let client = ManagementForeignKeyClient(api: try makeAPIClient())

        let page = try await client.search(endpoint: "not-a-resource", query: nil, cursor: nil)

        XCTAssertTrue(page.items.isEmpty)
        XCTAssertNil(page.nextCursor)
    }

    func testResolveFetchesSingleRecordByID() async throws {
        StubURLProtocol.install { request in
            XCTAssertEqual(request.url?.path, "/root/api/management/stations/s1")
            return try StubURLProtocol.response(
                for: request,
                json: #"{"id":"s1","name":"Tokyo"}"#
            )
        }
        let client = ManagementForeignKeyClient(api: try makeAPIClient())

        let item = try await client.resolve(endpoint: "stations", id: "s1")

        XCTAssertEqual(item?.id, "s1")
        XCTAssertEqual(item?.title, "Tokyo")
    }

    func testRecordMappingUsesTitleAndSubtitleHeuristics() {
        let record = TestFixtures.record(
            id: "r1", name: "Yamanote Line", additionalValues: ["nameEn": .string("Yamanote")])

        let item = ManagementForeignKeyClient.item(record)

        XCTAssertEqual(item?.id, "r1")
        XCTAssertEqual(item?.title, "Yamanote Line")
        XCTAssertEqual(item?.subtitle, "Yamanote")

        XCTAssertNil(ManagementForeignKeyClient.item(ResourceRecord(values: ["name": .string("no id")])))
    }

    private func makeAPIClient() throws -> APIClient {
        let configuration = try TestFixtures.configuration()
        return APIClient(
            configuration: configuration,
            auth: AuthSession(configuration: configuration),
            session: StubURLProtocol.session()
        )
    }
}
