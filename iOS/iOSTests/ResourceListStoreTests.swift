import XCTest
@testable import iOS

@MainActor
final class ResourceListStoreTests: XCTestCase {
    func testNewerSearchWinsWhenOlderResponseFinishesLast() async throws {
        let resource = try XCTUnwrap(ResourceDefinition.all.first { $0.id == "stations" })
        let api = ControlledResourceListAPI()
        let store = ResourceListStore(resource: resource)

        store.query = "old"
        let oldRequest = Task { await store.load(api: api) }
        let receivedOldRequest = await api.waitForRequest(query: "old")
        XCTAssertTrue(receivedOldRequest)

        store.query = "new"
        let newRequest = Task { await store.load(api: api) }
        let receivedNewRequest = await api.waitForRequest(query: "new")
        XCTAssertTrue(receivedNewRequest)

        api.resolve(query: "new", page: page(named: "New Result"))
        await newRequest.value
        XCTAssertEqual(store.items.map(\.title), ["New Result"])

        api.resolve(query: "old", page: page(named: "Stale Result"))
        await oldRequest.value
        XCTAssertEqual(store.items.map(\.title), ["New Result"])
        XCTAssertFalse(store.isLoading)
    }

    func testPaginationFromPreviousQueryCannotAppendToNewResults() async throws {
        let resource = try XCTUnwrap(ResourceDefinition.all.first { $0.id == "stations" })
        let api = ControlledResourceListAPI()
        let store = ResourceListStore(resource: resource)

        store.query = "old"
        let initialRequest = Task { await store.load(api: api) }
        let receivedInitialRequest = await api.waitForRequest(query: "old")
        XCTAssertTrue(receivedInitialRequest)
        api.resolve(query: "old", page: page(named: "Old First", next: "page-2"))
        await initialRequest.value

        let paginationRequest = Task { await store.load(api: api, reset: false) }
        let receivedPaginationRequest = await api.waitForRequest(query: "old", cursor: "page-2")
        XCTAssertTrue(receivedPaginationRequest)

        store.query = "new"
        let newRequest = Task { await store.load(api: api) }
        let receivedNewRequest = await api.waitForRequest(query: "new")
        XCTAssertTrue(receivedNewRequest)
        api.resolve(query: "new", page: page(named: "New First"))
        await newRequest.value

        api.resolve(query: "old", cursor: "page-2", page: page(named: "Stale Second"))
        await paginationRequest.value
        XCTAssertEqual(store.items.map(\.title), ["New First"])
        XCTAssertNil(store.nextCursor)
    }

    private func page(named name: String, next: String? = nil) -> ResourcePage {
        ResourcePage(
            items: [TestFixtures.record(id: name, name: name)],
            pagination: Pagination(next: next)
        )
    }
}

@MainActor
private final class ControlledResourceListAPI: ResourceListAPI {
    private struct RequestKey: Hashable {
        let query: String
        let cursor: String?
    }

    private var pending: [RequestKey: CheckedContinuation<ResourcePage, any Error>] = [:]

    func listPage(
        _: ResourceDefinition,
        query: String,
        cursor: String?
    ) async throws -> ResourcePage {
        try await withCheckedThrowingContinuation { continuation in
            pending[RequestKey(query: query, cursor: cursor)] = continuation
        }
    }

    func waitForRequest(query: String, cursor: String? = nil) async -> Bool {
        let key = RequestKey(query: query, cursor: cursor)
        for _ in 0 ..< 100 where pending[key] == nil {
            await Task.yield()
        }
        return pending[key] != nil
    }

    func resolve(query: String, cursor: String? = nil, page: ResourcePage) {
        let key = RequestKey(query: query, cursor: cursor)
        pending.removeValue(forKey: key)?.resume(returning: page)
    }
}
