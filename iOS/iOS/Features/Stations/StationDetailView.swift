import Observation
import SwiftUI

@MainActor
@Observable
private final class StationRelationsStore {
    var codes: [ResourceRecord] = []
    var platforms: [ResourceRecord] = []
    var routeStations: [ResourceRecord] = []

    func load(stationID: String, api: APIClient) async {
        async let codesPage = load("station-codes", filter: "stationId", id: stationID, api: api)
        async let platformPage = load("platforms", filter: "stationId", id: stationID, api: api)
        async let routePage = load("route-stations", filter: "stationId", id: stationID, api: api)
        (codes, platforms, routeStations) = await (codesPage, platformPage, routePage)
    }

    private func load(_ resourceID: String, filter: String, id: String, api: APIClient) async -> [ResourceRecord] {
        guard let resource = ResourceDefinition.all.first(where: { $0.id == resourceID }) else { return [] }
        return (try? await api.list(resource, filters: [filter: id]).items) ?? []
    }
}

struct StationDetailView: View {
    @Environment(AppDependencies.self) private var dependencies
    let resource: ResourceDefinition
    let record: ResourceRecord
    let onDeleted: () -> Void
    let showsActions: Bool
    @State private var relations = StationRelationsStore()

    init(
        resource: ResourceDefinition,
        record: ResourceRecord,
        showsActions: Bool = true,
        onDeleted: @escaping () -> Void
    ) {
        self.resource = resource
        self.record = record
        self.onDeleted = onDeleted
        self.showsActions = showsActions
    }

    var body: some View {
        ResourceDetailView(
            resource: resource,
            record: record,
            header: AnyView(header),
            showsActions: showsActions,
            onDeleted: onDeleted
        )
        .task {
            if let id = record.stableID { await relations.load(stationID: id, api: dependencies.api) }
        }
    }

    private var header: some View {
        VStack(spacing: 12) {
            if let latitude = record.latitude, let longitude = record.longitude {
                MapLibreView(
                    configuration: dependencies.configuration,
                    savedStations: [record],
                    candidates: [],
                    focus: .init(latitude: latitude, longitude: longitude),
                    initialBounds: nil,
                    onBoundsChanged: { _ in },
                    onMapTap: { _, _ in },
                    onCandidateTap: { _, _ in },
                    onSavedStationTap: { _ in }
                )
                .frame(height: 240)
                .clipShape(.rect(cornerRadius: 20))
            }
            HStack(spacing: 12) {
                RelationCount(title: "Codes", count: relations.codes.count, icon: "number")
                RelationCount(
                    title: "Routes",
                    count: relations.routeStations.count,
                    icon: "point.topleft.down.curvedto.point.bottomright.up"
                )
                RelationCount(title: "Platforms", count: relations.platforms.count, icon: "rectangle.split.3x1")
            }
        }
    }
}

private struct RelationCount: View {
    let title: String
    let count: Int
    let icon: String
    var body: some View {
        VStack(spacing: 4) {
            Image(systemName: icon).foregroundStyle(RailwayTheme.accent)
            Text(count.formatted()).font(.headline)
            Text(title).font(.caption).foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
        .railwayGlassCard()
    }
}
