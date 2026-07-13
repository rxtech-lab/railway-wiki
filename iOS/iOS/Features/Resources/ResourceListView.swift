import SwiftUI

struct ResourceListView: View {
    @Environment(AppDependencies.self) private var dependencies
    let resource: ResourceDefinition
    let compact: Bool
    let onSelect: (ResourceRecord) -> Void
    @State private var store: ResourceListStore
    @State private var presentsCreate = false

    init(
        resource: ResourceDefinition,
        compact: Bool,
        onSelect: @escaping (ResourceRecord) -> Void = { _ in }
    ) {
        self.resource = resource
        self.compact = compact
        self.onSelect = onSelect
        _store = State(initialValue: ResourceListStore(resource: resource))
    }

    var body: some View {
        @Bindable var store = store
        listView
        .scrollDismissesKeyboard(.interactively)
        .overlay {
            if store.items.isEmpty, !store.isLoading, store.errorMessage == nil {
                ContentUnavailableView(
                    "No \(resource.plural)",
                    systemImage: resource.icon,
                    description: Text("Create the first \(resource.singular.lowercased()) to get started.")
                )
            }
        }
        .navigationTitle(resource.plural)
        .resourceSearchable(text: $store.query, resource: resource)
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button("Add \(resource.singular)", systemImage: "plus") { presentsCreate = true }
                    .accessibilityIdentifier("resource.\(resource.id).add")
            }
            ToolbarItem(placement: .secondaryAction) {
                Button("Refresh", systemImage: "arrow.clockwise") {
                    Task { await store.load(api: dependencies.api) }
                }
            }
        }
        .sheet(isPresented: $presentsCreate) {
            SchemaFormSheet(resource: resource, onSaved: { _ in
                Task { await store.load(api: dependencies.api) }
            })
        }
        .task {
            guard !resource.searchable, store.items.isEmpty else { return }
            await store.load(api: dependencies.api)
        }
        .task(id: store.query) {
            guard resource.searchable else { return }
            if !store.query.isEmpty {
                try? await Task.sleep(for: .milliseconds(250))
            }
            guard !Task.isCancelled else { return }
            await store.load(api: dependencies.api)
        }
        // Derived from the STORE's resource (not the view prop) so it doubles as a
        // staleness probe: if a reused/stale store backs this list (the missing-.id
        // bug), the id reflects the previous resource, not the one now selected.
        // For fresh navigation store.resource == resource, so existing tests that
        // query "resource.<id>.list" are unaffected.
        .accessibilityIdentifier("resource.\(store.resource.id).list")
    }

    // Regular (multi-column) mode drives selection through the List so the
    // chosen row shows the standard highlighted/selected state; compact mode
    // pushes onto the NavigationStack instead.
    @ViewBuilder
    private var listView: some View {
        if compact {
            List { listRows }
        } else {
            List(selection: selectionBinding) { listRows }
        }
    }

    @ViewBuilder
    private var listRows: some View {
        if let error = store.errorMessage, store.items.isEmpty {
            ContentUnavailableView(
                "Could Not Load \(resource.plural)",
                systemImage: "exclamationmark.triangle",
                description: Text(error)
            )
        }
        ForEach(store.items) { item in
            row(for: item)
                .task { await store.loadNextIfNeeded(item: item, api: dependencies.api) }
        }
        if store.isLoading, !store.items.isEmpty {
            HStack { Spacer(); ProgressView(); Spacer() }
        }
    }

    // Bridges the List's selection to the shared navigation record. The getter
    // reflects whatever record is currently open in the detail column so the
    // row stays highlighted; the setter maps the selected id back to its record.
    private var selectionBinding: Binding<String?> {
        Binding(
            get: { dependencies.navigation.selectedRecord?.id },
            set: { newValue in
                guard let newValue,
                      let item = store.items.first(where: { $0.id == newValue }) else { return }
                onSelect(item)
            }
        )
    }

    @ViewBuilder
    private func row(for item: ResourceRecord) -> some View {
        if compact, let id = item.stableID {
            NavigationLink(value: CompactRoute.record(resourceID: resource.id, recordID: id)) {
                ResourceRow(item: item, icon: resource.icon)
            }
        } else {
            ResourceRow(item: item, icon: resource.icon)
        }
    }
}

private extension View {
    @ViewBuilder
    func resourceSearchable(text: Binding<String>, resource: ResourceDefinition) -> some View {
        if resource.searchable {
            searchable(text: text, prompt: "Search \(resource.plural)")
        } else {
            self
        }
    }
}

private struct ResourceRow: View {
    let item: ResourceRecord
    let icon: String

    var body: some View {
        Label {
            VStack(alignment: .leading, spacing: 2) {
                Text(item.title).foregroundStyle(.primary)
                if let subtitle = item.subtitle {
                    Text(subtitle).font(.caption).foregroundStyle(.secondary).lineLimit(1)
                }
            }
        } icon: {
            Image(systemName: icon).foregroundStyle(RailwayTheme.accent)
        }
        .accessibilityElement(children: .combine)
        .accessibilityIdentifier("record.\(item.stableID ?? "unknown")")
    }
}
