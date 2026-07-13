import SwiftUI

struct SidebarView: View {
    @Environment(AppDependencies.self) private var dependencies
    @Binding var selection: SidebarDestination

    var body: some View {
        List(selection: Binding<SidebarDestination?>(
            get: { selection },
            set: { if let value = $0 { selection = value } }
        )) {
            Section {
                Label("Dashboard", systemImage: "square.grid.2x2")
                    .tag(SidebarDestination.dashboard)
                    .accessibilityElement(children: .combine)
                    .accessibilityIdentifier("sidebar.dashboard")
                VStack(alignment: .leading, spacing: 6) {
                    Label("Map", systemImage: "map")
                    if selection == .map {
                        mapStatus
                            .foregroundStyle(.secondary)
                    }
                }
                    .tag(SidebarDestination.map)
                    .accessibilityElement(children: .contain)
                    .accessibilityIdentifier("sidebar.map")
            }
            ForEach(ResourceGroup.allCases) { group in
                Section(group.rawValue) {
                    ForEach(ResourceDefinition.all.filter { $0.group == group }) { resource in
                        Label(resource.plural, systemImage: resource.icon)
                            .tag(SidebarDestination.resource(resource.id))
                            .accessibilityElement(children: .combine)
                            .accessibilityIdentifier("sidebar.\(resource.id)")
                    }
                }
            }
        }
        .navigationTitle("Railway Wiki")
        .listStyle(.sidebar)
    }

    @ViewBuilder
    private var mapStatus: some View {
        let store = dependencies.mapStore
        if store.isSearching {
            MapStatusBadge(title: "Searching railway data…", icon: "network", identifier: "map.searching")
        } else if !store.candidates.isEmpty {
            MapStatusBadge(
                title: "\(store.candidates.count) railway candidates",
                icon: "mappin.and.ellipse",
                identifier: "map.candidates"
            )
        } else if store.isViewportSearchable {
            MapStatusBadge(title: "Ready to search this area", icon: "checkmark", identifier: "map.searchReady")
        } else {
            MapStatusBadge(title: "Zoom in to discover railway data", icon: "plus.magnifyingglass", identifier: "map.zoomHint")
        }
    }
}

struct CompactSidebarView: View {
    var body: some View {
        List {
            Section {
                NavigationLink(value: CompactRoute.destination(.dashboard)) {
                    Label("Dashboard", systemImage: "square.grid.2x2")
                }
                .accessibilityElement(children: .combine)
                .accessibilityIdentifier("sidebar.dashboard")
                NavigationLink(value: CompactRoute.destination(.map)) {
                    Label("Map", systemImage: "map")
                }
                .accessibilityElement(children: .combine)
                .accessibilityIdentifier("sidebar.map")
            }
            ForEach(ResourceGroup.allCases) { group in
                Section(group.rawValue) {
                    ForEach(ResourceDefinition.all.filter { $0.group == group }) { resource in
                        NavigationLink(value: CompactRoute.destination(.resource(resource.id))) {
                            Label(resource.plural, systemImage: resource.icon)
                        }
                        .accessibilityElement(children: .combine)
                        .accessibilityIdentifier("sidebar.\(resource.id)")
                    }
                }
            }
        }
        .navigationTitle("Railway Wiki")
    }
}
