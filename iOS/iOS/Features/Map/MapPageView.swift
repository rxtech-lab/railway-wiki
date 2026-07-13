import SwiftUI

struct MapPageView: View {
    private static let leadingPanelWidth: CGFloat = 360
    private static let leadingPanelOuterPadding: CGFloat = 24

    @Environment(AppDependencies.self) private var dependencies
    @StateObject private var locationProvider = StationLocationProvider()
    @State private var locationFocus: MapCoordinate?
    @State private var locationFocusRequestID = 0
    @State private var isLocationCentered = false
    @State private var candidateSheetDetent: MapCandidatePanelDetent = .collapsed
    @State private var mapCandidatePopoverAnchor: CGPoint?
    @State private var mapPopoverCandidate: OverpassCandidate?
    @State private var isImportingCandidate = false
    @State private var importedCandidateName: String?
    @State private var selectedSavedStation: ResourceRecord?

    var body: some View {
        GeometryReader { proxy in
            let usesLeadingPanel = proxy.size.width >= 700 || proxy.size.width > proxy.size.height
            adaptiveMap(
                usesLeadingPanel: usesLeadingPanel,
                cameraLeadingInset: usesLeadingPanel
                    ? proxy.safeAreaInsets.leading + Self.leadingPanelWidth + Self.leadingPanelOuterPadding
                    : 0
            )
        }
        .navigationTitle("")
        .navigationBarTitleDisplayMode(.inline)
        .toolbarBackground(.hidden, for: .navigationBar)
        .toolbar {
            ToolbarItemGroup(placement: .primaryAction) {
                Button(
                    "Current Location",
                    systemImage: isLocationCentered ? "location.fill" : "location"
                ) {
                    centerOnCurrentLocation()
                }
                .foregroundStyle(isLocationCentered ? RailwayTheme.accent : Color.primary)
                .accessibilityIdentifier("map.currentLocation")
                .accessibilityValue(isLocationCentered ? "Centered" : "Not centered")

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
        .overlay {
            if isImportingCandidate {
                ZStack {
                    Color.black.opacity(0.18)
                        .ignoresSafeArea()
                    ProgressView("Adding station…")
                        .padding(.horizontal, 24)
                        .padding(.vertical, 18)
                        .background(.regularMaterial, in: .rect(cornerRadius: 16))
                }
                .accessibilityElement(children: .combine)
                .accessibilityIdentifier("map.candidate.importing")
            }
        }
        .alert("Map Action Failed", isPresented: errorBinding) {
            Button("OK") { store.errorMessage = nil }
        } message: { Text(store.errorMessage ?? "") }
        .alert("Location Unavailable", isPresented: locationErrorBinding) {
            Button("OK") { locationProvider.clearError() }
        } message: { Text(locationProvider.errorMessage ?? "") }
        .alert("Station Imported", isPresented: importSuccessBinding) {
            Button("OK") { importedCandidateName = nil }
        } message: {
            Text("\(importedCandidateName ?? "The station") was successfully imported.")
        }
        .sheet(item: $selectedSavedStation) { station in
            SavedStationMapSheet(station: station)
        }
        .onAppear { centerOnCurrentLocation() }
        .onChange(of: locationProvider.coordinate) { _, coordinate in
            guard isLocationCentered, let coordinate else { return }
            recenterMap(on: coordinate)
        }
        .onChange(of: locationProvider.errorMessage) { _, errorMessage in
            if errorMessage != nil {
                isLocationCentered = false
            }
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
    private func adaptiveMap(usesLeadingPanel: Bool, cameraLeadingInset: CGFloat) -> some View {
        if usesLeadingPanel {
            ZStack(alignment: .leading) {
                immersiveMap(
                    showsCandidatePopover: true,
                    cameraLeadingInset: cameraLeadingInset
                )

                MapCandidatePanel(compact: false)
                    .frame(width: Self.leadingPanelWidth)
                    .padding(12)
                    .popover(item: selectedCandidateBinding) { candidate in
                        candidatePopover(candidate)
                    }

                Color.clear
                    .frame(width: 1, height: 1)
                    .position(mapCandidatePopoverAnchor ?? CGPoint(x: -1, y: -1))
                    .allowsHitTesting(false)
                    .popover(
                        item: $mapPopoverCandidate,
                        attachmentAnchor: .point(.center),
                        arrowEdge: .bottom
                    ) { candidate in
                        candidatePopover(candidate)
                    }
            }
        } else {
            immersiveMap(showsCandidatePopover: false, cameraLeadingInset: 0)
                .overlay(alignment: .bottom) {
                    CompactMapCandidateSheet(detent: $candidateSheetDetent) {
                        MapCandidatePanel(compact: true)
                    }
                }
        }
    }

    private func immersiveMap(showsCandidatePopover: Bool, cameraLeadingInset: CGFloat) -> some View {
        MapLibreView(
            configuration: dependencies.configuration,
            savedStations: store.savedStations,
            candidates: store.candidates,
            focus: locationFocus ?? (store.bounds == nil ? store.initialFocus : nil),
            focusRequestID: locationFocusRequestID,
            initialBounds: store.bounds,
            cameraPadding: UIEdgeInsets(top: 0, left: cameraLeadingInset, bottom: 0, right: 0),
            showsUserLocation: true,
            onBoundsChanged: { store.updateBounds($0, api: dependencies.api) },
            onCameraInteraction: { isLocationCentered = false },
            onMapTap: nil,
            onCandidateTap: { candidate, anchor in
                if showsCandidatePopover {
                    mapPopoverCandidate = nil
                    mapCandidatePopoverAnchor = anchor
                    Task { @MainActor in
                        await Task.yield()
                        mapPopoverCandidate = candidate
                    }
                } else {
                    dismissMapPopover()
                    store.selectedCandidate = candidate
                }
            },
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

    private func candidatePopover(_ candidate: OverpassCandidate) -> some View {
        CandidateQuickAddView(
            candidate: candidate,
            add: { importCandidate(candidate) },
            review: { review(candidate) },
            close: { dismissCandidate() }
        )
        .presentationCompactAdaptation(.popover)
    }

    private func clearCandidates() {
        store.candidates = []
        dismissCandidate()
        candidateSheetDetent = .collapsed
    }

    private func review(_ candidate: OverpassCandidate) {
        dismissCandidate()
        store.reviewCandidate = candidate
    }

    private func importCandidate(_ candidate: OverpassCandidate) {
        guard !isImportingCandidate else { return }
        dismissCandidate()
        isImportingCandidate = true
        Task {
            let succeeded = await store.importCandidate(candidate, api: dependencies.api)
            isImportingCandidate = false
            if succeeded {
                importedCandidateName = candidate.name
            }
        }
    }

    private func dismissCandidate() {
        store.selectedCandidate = nil
        dismissMapPopover()
    }

    private func dismissMapPopover() {
        mapPopoverCandidate = nil
        mapCandidatePopoverAnchor = nil
    }

    private func centerOnCurrentLocation() {
        isLocationCentered = true
        if let coordinate = locationProvider.coordinate {
            recenterMap(on: coordinate)
        }
        locationProvider.requestCurrentLocation()
    }

    private func recenterMap(on coordinate: MapCoordinate) {
        locationFocus = coordinate
        locationFocusRequestID &+= 1
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

    private var importSuccessBinding: Binding<Bool> {
        Binding(
            get: { importedCandidateName != nil },
            set: { if !$0 { importedCandidateName = nil } }
        )
    }

}

private extension MapPageView {
    func openSavedStation(_ id: String) {
        selectedSavedStation = store.savedStations.first { $0.stableID == id }
    }
}
