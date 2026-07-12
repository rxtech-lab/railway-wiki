import SwiftUI

struct MapPageView: View {
    @Environment(AppDependencies.self) private var dependencies
    @StateObject private var locationProvider = StationLocationProvider()
    @State private var locationFocus: MapCoordinate?
    @State private var candidateSheetDetent: MapCandidatePanelDetent = .collapsed

    var body: some View {
        GeometryReader { proxy in
            adaptiveMap(usesLeadingPanel: proxy.size.width >= 700 || proxy.size.width > proxy.size.height)
        }
        .navigationTitle("")
        .navigationBarTitleDisplayMode(.inline)
        .toolbarBackground(.hidden, for: .navigationBar)
        .toolbar {
            ToolbarItemGroup(placement: .primaryAction) {
                Button("Current Location", systemImage: "location") {
                    locationProvider.requestCurrentLocation()
                }
                .accessibilityIdentifier("map.currentLocation")

                Button("Search This Area", systemImage: "magnifyingglass") {
                    Task { await store.searchViewport(api: dependencies.api) }
                }
                .disabled(store.isSearching || !store.isViewportSearchable)
                .accessibilityIdentifier("map.searchViewport")

                Menu {
                    Button("Clear Candidates", systemImage: "xmark.circle") {
                        clearCandidates()
                    }
                    Button("Reload Saved Stations", systemImage: "arrow.clockwise") {
                        Task { await store.loadSaved(api: dependencies.api) }
                    }
                } label: {
                    Label("More", systemImage: "ellipsis")
                }
                .accessibilityIdentifier("map.more")
            }
        }
        .alert("Map Action Failed", isPresented: errorBinding) {
            Button("OK") { store.errorMessage = nil }
        } message: { Text(store.errorMessage ?? "") }
        .alert("Location Unavailable", isPresented: locationErrorBinding) {
            Button("OK") { locationProvider.clearError() }
        } message: { Text(locationProvider.errorMessage ?? "") }
        .onAppear { locationProvider.requestCurrentLocation() }
        .onChange(of: locationProvider.coordinate) { _, coordinate in
            guard let coordinate else { return }
            locationFocus = coordinate
        }
        .onChange(of: store.candidates.count) { oldCount, newCount in
            guard newCount > oldCount else { return }
            candidateSheetDetent = .medium
        }
        .onChange(of: store.selectedCandidate?.id) { _, candidateID in
            guard candidateID != nil else { return }
            candidateSheetDetent = .medium
        }
    }

    private var store: MapStore { dependencies.mapStore }

    @ViewBuilder
    private func adaptiveMap(usesLeadingPanel: Bool) -> some View {
        if usesLeadingPanel {
            immersiveMap
                .overlay(alignment: .leading) {
                    MapCandidatePanel(compact: false)
                        .frame(width: 360)
                        .padding(12)
                        .popover(item: selectedCandidateBinding) { candidate in
                            CandidateQuickAddView(
                                candidate: candidate,
                                add: { Task { await store.importCandidate(candidate, api: dependencies.api) } },
                                review: { review(candidate) },
                                close: { store.selectedCandidate = nil }
                            )
                            .presentationCompactAdaptation(.popover)
                        }
                }
        } else {
            immersiveMap
                .overlay(alignment: .bottom) {
                    CompactMapCandidateSheet(detent: $candidateSheetDetent) {
                        MapCandidatePanel(compact: true)
                    }
                }
        }
    }

    private var immersiveMap: some View {
        MapLibreView(
            configuration: dependencies.configuration,
            savedStations: store.savedStations,
            candidates: store.candidates,
            focus: locationFocus ?? (store.bounds == nil ? store.initialFocus : nil),
            initialBounds: store.bounds,
            showsUserLocation: true,
            onBoundsChanged: { store.updateBounds($0, api: dependencies.api) },
            onMapTap: nil,
            onCandidateTap: { store.selectedCandidate = $0 },
            onSavedStationTap: openSavedStation
        )
        .accessibilityIdentifier("map.page")
        .ignoresSafeArea()
    }

    private var selectedCandidateBinding: Binding<OverpassCandidate?> {
        Binding(
            get: { store.selectedCandidate },
            set: { store.selectedCandidate = $0 }
        )
    }

    private func clearCandidates() {
        store.candidates = []
        store.selectedCandidate = nil
        candidateSheetDetent = .collapsed
    }

    private func review(_ candidate: OverpassCandidate) {
        store.selectedCandidate = nil
        store.reviewCandidate = candidate
    }

    private var errorBinding: Binding<Bool> {
        Binding(
            get: { store.errorMessage != nil },
            set: { if !$0 { store.errorMessage = nil } }
        )
    }

    private var locationErrorBinding: Binding<Bool> {
        Binding(
            get: { locationProvider.errorMessage != nil },
            set: { if !$0 { locationProvider.clearError() } }
        )
    }

    private func openSavedStation(_ id: String) {
        guard let resource = ResourceDefinition.all.first(where: { $0.id == "stations" }) else { return }
        Task {
            if let station = try? await dependencies.api.get(resource, id: id) {
                dependencies.navigation.select(.resource(resource.id))
                dependencies.navigation.select(station)
            }
        }
    }
}
