import Foundation
import Observation
import SwiftUI

@MainActor
@Observable
private final class DashboardStore {
    var snapshot: DashboardSnapshot?
    var stations: [ResourceRecord] = []
    var isLoading = false
    var errorMessage: String?

    func load(api: APIClient) async {
        isLoading = true
        defer { isLoading = false }
        do {
            snapshot = try await api.dashboard()
            if let stationResource = ResourceDefinition.all.first(where: { $0.id == "stations" }) {
                stations = try await api.list(stationResource).items
            }
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

struct DashboardView: View {
    @Environment(AppDependencies.self) private var dependencies
    @State private var store = DashboardStore()

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 20) {
                if let snapshot = store.snapshot {
                    mapPreview
                    Label("Railway API Online", systemImage: "checkmark.circle.fill")
                        .foregroundStyle(.green)
                        .railwayGlassCard()
                    coverageCard(snapshot.stationCoverage)
                    resourceGrid(snapshot.resources)
                    Text("Updated \(snapshot.generatedAt.formatted(date: .abbreviated, time: .shortened))")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else if let error = store.errorMessage {
                    ContentUnavailableView(
                        "Dashboard Unavailable",
                        systemImage: "exclamationmark.triangle",
                        description: Text(error)
                    )
                }
            }
            .padding()
        }
        .navigationTitle("Dashboard")
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button("Refresh", systemImage: "arrow.clockwise") {
                    Task { await store.load(api: dependencies.api) }
                }
            }
        }
        .modifier(LoadingOverlay(visible: store.isLoading))
        .task { if store.snapshot == nil { await store.load(api: dependencies.api) } }
        .accessibilityIdentifier("dashboard")
    }

    private var mapPreview: some View {
        let mappedStations = store.stations.filter { $0.latitude != nil && $0.longitude != nil }
        let focus = mappedStations.first.flatMap { station -> MapCoordinate? in
            guard let latitude = station.latitude, let longitude = station.longitude else { return nil }
            return .init(latitude: latitude, longitude: longitude)
        }
        return MapLibreView(
            configuration: dependencies.configuration,
            savedStations: mappedStations,
            candidates: [],
            focus: focus,
            initialBounds: nil,
            onBoundsChanged: { _ in },
            onMapTap: { _, _ in },
            onCandidateTap: { _ in },
            onSavedStationTap: openStation
        )
        .frame(height: 260)
        .clipShape(.rect(cornerRadius: 22))
        .accessibilityIdentifier("dashboard.map")
    }

    private func openStation(_ id: String) {
        guard let station = store.stations.first(where: { $0.stableID == id }) else { return }
        dependencies.navigation.select(.resource("stations"))
        dependencies.navigation.select(station)
    }

    private func coverageCard(_ coverage: DashboardStationCoverage) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Label("Station Coordinate Coverage", systemImage: "location.fill")
                .font(.headline)
            ProgressView(value: Double(coverage.withCoordinates), total: Double(max(coverage.total, 1)))
            Text("\(coverage.withCoordinates) of \(coverage.total) stations can be shown on the map")
                .font(.subheadline)
                .foregroundStyle(.secondary)
        }
        .railwayGlassCard()
    }

    private func resourceGrid(_ metrics: [DashboardResourceMetric]) -> some View {
        LazyVGrid(columns: [GridItem(.adaptive(minimum: 150), spacing: 12)], spacing: 12) {
            ForEach(metrics) { metric in
                VStack(alignment: .leading, spacing: 6) {
                    Text(metric.resource.replacingOccurrences(of: "-", with: " ").capitalized)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Text(metric.count.formatted())
                        .font(.title2.bold())
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .railwayGlassCard()
            }
        }
    }
}
