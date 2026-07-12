import Foundation
import Observation

@MainActor
protocol ResourceListAPI: AnyObject {
    func listPage(
        _ resource: ResourceDefinition,
        query: String,
        cursor: String?
    ) async throws -> ResourcePage
}

extension APIClient: ResourceListAPI {
    func listPage(
        _ resource: ResourceDefinition,
        query: String,
        cursor: String?
    ) async throws -> ResourcePage {
        try await list(resource, query: query, cursor: cursor)
    }
}

@MainActor
@Observable
final class ResourceListStore {
    let resource: ResourceDefinition
    var items: [ResourceRecord] = []
    var query = ""
    var nextCursor: String?
    var isLoading = false
    var errorMessage: String?
    private var requestGeneration = 0

    init(resource: ResourceDefinition) { self.resource = resource }

    func load(api: any ResourceListAPI, reset: Bool = true) async {
        let requestedCursor: String?
        if reset {
            requestGeneration &+= 1
            requestedCursor = nil
        } else {
            guard !isLoading, let nextCursor else { return }
            requestedCursor = nextCursor
        }

        let generation = requestGeneration
        let requestedQuery = effectiveQuery
        isLoading = true
        defer {
            if generation == requestGeneration {
                isLoading = false
            }
        }

        do {
            let page = try await api.listPage(resource, query: requestedQuery, cursor: requestedCursor)
            guard !Task.isCancelled,
                  generation == requestGeneration,
                  requestedQuery == effectiveQuery
            else { return }
            items = reset ? page.items : items + page.items
            nextCursor = page.pagination.next
            errorMessage = nil
        } catch is CancellationError {
            return
        } catch {
            guard generation == requestGeneration, requestedQuery == effectiveQuery else { return }
            errorMessage = error.localizedDescription
        }
    }

    func loadNextIfNeeded(item: ResourceRecord, api: any ResourceListAPI) async {
        guard item.stableID == items.last?.stableID, nextCursor != nil else { return }
        await load(api: api, reset: false)
    }

    private var effectiveQuery: String {
        guard resource.searchable else { return "" }
        return query.trimmingCharacters(in: .whitespacesAndNewlines)
    }
}
