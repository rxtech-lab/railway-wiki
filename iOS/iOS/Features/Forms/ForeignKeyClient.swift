import Foundation
import JSONSchemaForm

/// Foreign-key search client backing `ui:widget: "foreign-key"` fields.
///
/// The server's `ui:options.endpoint` carries a management resource id
/// (e.g. "stations"); this client maps it to the matching
/// `ResourceDefinition` and reuses `APIClient`'s authenticated, cursor-
/// paginated list/get endpoints.
@MainActor
final class ManagementForeignKeyClient: ForeignKeySearchClient {
    private let api: APIClient

    init(api: APIClient) {
        self.api = api
    }

    func search(endpoint: String, query: String?, cursor: String?) async throws -> ForeignKeyPage {
        guard let resource = Self.resource(for: endpoint) else {
            return ForeignKeyPage(items: [], nextCursor: nil)
        }
        let page = try await api.list(resource, query: query ?? "", cursor: cursor)
        return ForeignKeyPage(
            items: page.items.compactMap(Self.item),
            nextCursor: page.pagination.next
        )
    }

    func resolve(endpoint: String, id: String) async throws -> ForeignKeyItem? {
        guard let resource = Self.resource(for: endpoint) else { return nil }
        return Self.item(try await api.get(resource, id: id))
    }

    private static func resource(for endpoint: String) -> ResourceDefinition? {
        ResourceDefinition.all.first { $0.id == endpoint }
    }

    static func item(_ record: ResourceRecord) -> ForeignKeyItem? {
        guard let id = record.stableID else { return nil }
        return ForeignKeyItem(
            id: id,
            title: record.title,
            subtitle: record.subtitle,
            raw: JSONValue.object(record.values).formData
        )
    }
}
