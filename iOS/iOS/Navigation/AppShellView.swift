import Foundation
import RxAuthSwift
import SwiftUI

struct AppShellView: View {
    @Environment(AppDependencies.self) private var dependencies
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass

    var body: some View {
        Group {
            if horizontalSizeClass == .compact {
                CompactAppNavigation()
            } else {
                RegularAppNavigation()
            }
        }
        .onChange(of: horizontalSizeClass) { _, sizeClass in
            if sizeClass == .compact {
                dependencies.navigation.synchronizeForCompact()
            } else {
                dependencies.navigation.synchronizeForRegular()
            }
        }
    }
}

private struct RegularAppNavigation: View {
    @Environment(AppDependencies.self) private var dependencies

    var body: some View {
        @Bindable var navigation = dependencies.navigation
        Group {
            if navigation.selectedSidebar == .map || navigation.selectedSidebar == .dashboard {
                NavigationSplitView {
                    SidebarView(selection: $navigation.selectedSidebar)
                        .toolbar { accountToolbar }
                } detail: {
                    detailColumn
                }
                .navigationSplitViewStyle(.balanced)
            } else {
                NavigationSplitView {
                    SidebarView(selection: $navigation.selectedSidebar)
                        .toolbar { accountToolbar }
                } content: {
                    contentColumn
                } detail: {
                    detailColumn
                }
                .navigationSplitViewStyle(.balanced)
            }
        }
    }

    @ViewBuilder
    private var contentColumn: some View {
        switch dependencies.navigation.selectedSidebar {
        case .dashboard:
            SelectionSummary(title: "Dashboard", message: "System status and all resource totals", icon: "square.grid.2x2")
        case .map:
            EmptyView()
        case .resource(let id):
            if let resource = ResourceDefinition.all.first(where: { $0.id == id }) {
                ResourceListView(resource: resource, compact: false) { dependencies.navigation.select($0) }
                    // Give each resource a distinct view identity so the list's
                    // @State store is rebuilt on switch. Without this, SwiftUI
                    // reuses the identity and keeps serving the first resource's
                    // rows (e.g. Stations/Codes/Platforms showing Companies data).
                    .id(resource.id)
            }
        }
    }

    @ViewBuilder
    private var detailColumn: some View {
        switch dependencies.navigation.selectedSidebar {
        case .dashboard:
            DashboardView()
        case .map:
            MapPageView()
        case .resource(let id):
            if let resource = ResourceDefinition.all.first(where: { $0.id == id }),
               let record = dependencies.navigation.selectedRecord {
                SpecializedResourceDetail(resource: resource, record: record) {
                    dependencies.navigation.selectedRecord = nil
                }
                .id(record.stableID)
            } else {
                ContentUnavailableView(
                    "Select a Record",
                    systemImage: "sidebar.right",
                    description: Text("Choose a record from the list to inspect and edit it.")
                )
            }
        }
    }

    private var accountToolbar: some ToolbarContent {
        ToolbarItem(placement: .bottomBar) {
            Menu {
                if let user = dependencies.auth.manager.currentUser {
                    Text(user.email ?? user.name ?? user.id)
                }
                Button("Sign Out", systemImage: "rectangle.portrait.and.arrow.right") {
                    Task { await dependencies.auth.signOut() }
                }
            } label: {
                Label("Account", systemImage: "person.crop.circle")
            }
        }
    }
}

private struct CompactAppNavigation: View {
    @Environment(AppDependencies.self) private var dependencies

    var body: some View {
        @Bindable var navigation = dependencies.navigation
        NavigationStack(path: $navigation.compactPath) {
            CompactSidebarView()
                .navigationDestination(for: CompactRoute.self) { route in
                    switch route {
                    case .destination(let destination): compactDestination(destination)
                    case .record(let resourceID, let recordID):
                        CompactRecordLoader(resourceID: resourceID, recordID: recordID)
                    }
                }
                .toolbar {
                    ToolbarItem(placement: .topBarTrailing) {
                        Button("Sign Out", systemImage: "rectangle.portrait.and.arrow.right") {
                            Task { await dependencies.auth.signOut() }
                        }
                    }
                }
        }
        .onChange(of: navigation.compactPath) { _, _ in
            navigation.synchronizeForRegular()
        }
    }

    @ViewBuilder
    private func compactDestination(_ destination: SidebarDestination) -> some View {
        switch destination {
        case .dashboard: DashboardView()
        case .map: MapPageView()
        case .resource(let id):
            if let resource = ResourceDefinition.all.first(where: { $0.id == id }) {
                ResourceListView(resource: resource, compact: true)
            }
        }
    }
}

private struct CompactRecordLoader: View {
    @Environment(AppDependencies.self) private var dependencies
    let resourceID: String
    let recordID: String
    @State private var record: ResourceRecord?
    @State private var errorMessage: String?

    var body: some View {
        if let resource = ResourceDefinition.all.first(where: { $0.id == resourceID }) {
            if let record = displayedRecord {
                SpecializedResourceDetail(resource: resource, record: record)
            } else if let errorMessage {
                ContentUnavailableView(
                    "Record Unavailable",
                    systemImage: "exclamationmark.triangle",
                    description: Text(errorMessage)
                )
            } else {
                ProgressView().task { await load(resource) }
            }
        }
    }

    private func load(_ resource: ResourceDefinition) async {
        if displayedRecord != nil { return }
        do {
            record = try await dependencies.api.get(resource, id: recordID)
            if let record {
                dependencies.navigation.restoreCompactRecord(record, resourceID: resourceID)
            }
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private var displayedRecord: ResourceRecord? {
        if dependencies.navigation.selectedRecord?.stableID == recordID {
            return dependencies.navigation.selectedRecord
        }
        return record
    }
}

private struct SpecializedResourceDetail: View {
    let resource: ResourceDefinition
    let record: ResourceRecord
    let onDeleted: () -> Void

    init(resource: ResourceDefinition, record: ResourceRecord, onDeleted: @escaping () -> Void = {}) {
        self.resource = resource
        self.record = record
        self.onDeleted = onDeleted
    }

    var body: some View {
        switch resource.specialization {
        case .station: StationDetailView(resource: resource, record: record, onDeleted: onDeleted)
        case .route: RouteEditorView(resource: resource, record: record, onDeleted: onDeleted)
        case .generic: ResourceDetailView(resource: resource, record: record, onDeleted: onDeleted)
        }
    }
}

private struct SelectionSummary: View {
    let title: String
    let message: String
    let icon: String

    var body: some View {
        ContentUnavailableView { Label(title, systemImage: icon) } description: { Text(message) }
            .navigationTitle(title)
    }
}
