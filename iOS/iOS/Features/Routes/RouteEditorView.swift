import Foundation
import SwiftUI

struct RouteEditorView: View {
    @Environment(AppDependencies.self) private var dependencies
    let resource: ResourceDefinition
    let record: ResourceRecord
    let onDeleted: () -> Void
    @State private var presentsStationPicker = false
    @State private var confirmsDelete = false

    init(resource: ResourceDefinition, record: ResourceRecord, onDeleted: @escaping () -> Void = {}) {
        self.resource = resource
        self.record = record
        self.onDeleted = onDeleted
    }

    var body: some View {
        @Bindable var store = store
        Group {
            if let configuration = store.configuration {
                VStack(spacing: 0) {
                    routeHeader(configuration)
                    Divider()
                    HStack(spacing: 0) {
                        stationList(configuration)
                            .frame(minWidth: 260, idealWidth: 340, maxWidth: 420)
                        Divider()
                        inspector(configuration)
                    }
                }
            } else if let error = store.errorMessage {
                ContentUnavailableView("Route Unavailable", systemImage: "exclamationmark.triangle", description: Text(error))
            } else {
                ProgressView()
            }
        }
        .navigationTitle(configurationName)
        .toolbar { editorToolbar }
        .sheet(isPresented: $presentsStationPicker) {
            StationPickerSheet { store.addStation($0) }
        }
        .confirmationDialog("Delete this route?", isPresented: $confirmsDelete) {
            Button("Delete", role: .destructive) { Task { await deleteRoute() } }
            Button("Cancel", role: .cancel) {}
        }
        .alert("Route Editor", isPresented: Binding(
            get: { !store.validationErrors.isEmpty || store.errorMessage != nil },
            set: { if !$0 { store.validationErrors = []; store.errorMessage = nil } }
        )) {
            Button("OK") { store.validationErrors = []; store.errorMessage = nil }
        } message: {
            Text(store.errorMessage ?? store.validationErrors.joined(separator: "\n"))
        }
        .modifier(LoadingOverlay(visible: store.isLoading || store.isSaving))
        .task { await store.load(api: dependencies.api) }
        .accessibilityIdentifier("route.editor")
    }

    private var store: RouteEditorStore { dependencies.routeEditor(for: record) }

    private var configurationName: String { store.configuration?.route.title ?? record.title }

    private func routeHeader(_ configuration: RouteConfiguration) -> some View {
        HStack {
            Image(systemName: "point.topleft.down.curvedto.point.bottomright.up")
                .font(.title2).foregroundStyle(RailwayTheme.accent)
            TextField("Route name", text: Binding(
                get: { configuration.route.values["name"]?.stringValue ?? "" },
                set: { store.setRouteName($0) }
            ))
            .font(.title2.bold())
            Spacer()
            Text("\(configuration.stations.count) stations").foregroundStyle(.secondary)
        }
        .padding()
    }

    private func stationList(_ configuration: RouteConfiguration) -> some View {
        List(selection: Binding<String?>(get: { store.selectedStationID }, set: { store.selectedStationID = $0 })) {
            ForEach(Array(configuration.stations.enumerated()), id: \.element.stableID) { index, station in
                VStack(alignment: .leading, spacing: 3) {
                    Text(store.stationTitles[station.stationId] ?? station.stationId)
                    Text("Stop \(index + 1)" + distanceText(station)).font(.caption).foregroundStyle(.secondary)
                }
                .tag(station.stableID)
            }
            .onMove(perform: store.move)
        }
        .scrollDismissesKeyboard(.interactively)
        .environment(\.editMode, .constant(.active))
    }

    @ViewBuilder
    private func inspector(_ configuration: RouteConfiguration) -> some View {
        if let selected = configuration.stations.first(where: { $0.stableID == store.selectedStationID }) {
            RouteStationInspector(routeID: store.routeID, draft: selected) { store.updateStation($0) }
                .id(selected.stableID)
        } else {
            ContentUnavailableView(
                "Select a Station",
                systemImage: "slider.horizontal.3",
                description: Text("Choose an ordered station to edit its route metadata.")
            )
        }
    }

    @ToolbarContentBuilder
    private var editorToolbar: some ToolbarContent {
        ToolbarItemGroup(placement: .secondaryAction) {
            Button("Undo", systemImage: "arrow.uturn.backward", action: store.undo).disabled(!store.canUndo)
            Button("Redo", systemImage: "arrow.uturn.forward", action: store.redo).disabled(!store.canRedo)
        }
        ToolbarItem(placement: .secondaryAction) {
            Menu {
                Button("Add Station", systemImage: "plus") { presentsStationPicker = true }
                Button(
                    "Remove Selected Station",
                    systemImage: "minus.circle",
                    role: .destructive,
                    action: store.removeSelectedStation
                )
                Button("Validate", systemImage: "checkmark.shield") {
                    Task { await store.validate(api: dependencies.api) }
                }
                Divider()
                Button("Delete Route", systemImage: "trash", role: .destructive) { confirmsDelete = true }
            } label: { Label("More", systemImage: "ellipsis.circle") }
        }
        ToolbarItem(placement: .primaryAction) {
            Button("Save Changes", systemImage: "checkmark") {
                Task { await store.save(api: dependencies.api) }
            }
                .disabled(store.isSaving)
                .accessibilityIdentifier("route.save")
        }
    }

    private func distanceText(_ station: RouteStationDraft) -> String {
        station.distanceFromStartKm.map { " • \($0.formatted()) km" } ?? ""
    }

    private func deleteRoute() async {
        do {
            try await dependencies.api.delete(resource, id: store.routeID)
            dependencies.discardRouteEditor(id: store.routeID)
            onDeleted()
        } catch {
            store.errorMessage = error.localizedDescription
        }
    }
}
