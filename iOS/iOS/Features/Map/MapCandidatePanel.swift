import SwiftUI

struct MapCandidatePanel: View {
    @Environment(AppDependencies.self) private var dependencies
    let compact: Bool

    var body: some View {
        @Bindable var store = dependencies.mapStore
        Group {
            if compact, let candidate = store.selectedCandidate {
                ScrollView {
                    VStack(alignment: .leading, spacing: 12) {
                        Button("Results", systemImage: "chevron.left") {
                            store.selectedCandidate = nil
                        }
                        .accessibilityIdentifier("map.candidate.results")

                        CandidateQuickAddView(
                            candidate: candidate,
                            add: { Task { await store.importCandidate(candidate, api: dependencies.api) } },
                            review: {
                                store.selectedCandidate = nil
                                store.reviewCandidate = candidate
                            },
                            close: { store.selectedCandidate = nil },
                            showsCloseButton: false
                        )
                    }
                    .padding()
                }
            } else {
                VStack(spacing: 0) {
                    CandidatePanelHeader(
                        count: store.candidates.count,
                        search: compact ? {
                            Task { await store.searchViewport(api: dependencies.api) }
                        } : nil,
                        isSearchDisabled: store.isSearching || !store.isViewportSearchable,
                        clear: {
                            store.candidates = []
                            store.selectedCandidate = nil
                        }
                    )
                    MapCandidateList(store: store)
                }
            }
        }
        .background {
            ZStack {
                Color.clear
                    .accessibilityElement()
                    .accessibilityIdentifier("map.candidateList")
                if !compact {
                    RoundedRectangle(cornerRadius: 28)
                        .fill(.clear)
                        .glassEffect(.regular, in: .rect(cornerRadius: 28))
                }
            }
        }
        .clipShape(.rect(cornerRadius: compact ? 0 : 28))
        .sheet(item: $store.reviewCandidate) { candidate in
            if let station = ResourceDefinition.all.first(where: { $0.id == "stations" }) {
                SchemaFormSheet(
                    resource: station,
                    record: ResourceRecord(values: candidate.stationValues),
                    customSave: { values in
                        try await dependencies.api.importCandidate(candidate, overrides: values)
                    },
                    onSaved: { _ in
                        store.reviewCandidate = nil
                        Task { await store.loadSaved(api: dependencies.api) }
                    }
                )
            }
        }
    }
}

struct MapStatusBadge: View {
    let title: String
    let icon: String
    let identifier: String

    var body: some View {
        Label(title, systemImage: icon)
            .font(.subheadline)
            .lineLimit(1)
            .accessibilityIdentifier(identifier)
    }
}

struct CandidateQuickAddView: View {
    let candidate: OverpassCandidate
    let add: () -> Void
    let review: () -> Void
    let close: () -> Void
    var showsCloseButton = true

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            HStack(alignment: .top) {
                VStack(alignment: .leading) {
                    Text(candidate.name).font(.headline)
                    Text([candidate.mode, candidate.network].compactMap { $0 }.joined(separator: " • "))
                        .font(.caption).foregroundStyle(.secondary)
                }
                Spacer()
                if showsCloseButton {
                    Button("Close", systemImage: "xmark", action: close).labelStyle(.iconOnly)
                }
            }
            if candidate.importedStationId != nil {
                Label("Already in Railway Wiki", systemImage: "checkmark.circle.fill").foregroundStyle(.green)
            } else {
                Button("Add Station", systemImage: "plus.circle.fill", action: add)
                    .buttonStyle(.borderedProminent)
                    .accessibilityIdentifier("map.candidate.add")
                Button("Review & Edit", systemImage: "pencil", action: review)
                    .accessibilityIdentifier("map.candidate.review")
            }
        }
        .padding()
        .frame(idealWidth: 320)
    }
}

private struct CandidatePanelHeader: View {
    let count: Int
    let search: (() -> Void)?
    let isSearchDisabled: Bool
    let clear: () -> Void

    var body: some View {
        HStack {
            VStack(alignment: .leading, spacing: 2) {
                Text("Railway Candidates")
                    .font(.headline)
                Text(count == 1 ? "1 result" : "\(count) results")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            Spacer()
            if let search {
                Button("Search This Area", systemImage: "magnifyingglass", action: search)
                    .labelStyle(.iconOnly)
                    .disabled(isSearchDisabled)
                    .accessibilityIdentifier("map.candidates.searchViewport")
            }
            Button("Clear Candidates", systemImage: "xmark.circle", action: clear)
                .labelStyle(.iconOnly)
                .disabled(count == 0)
                .accessibilityIdentifier("map.candidates.clear")
        }
        .padding(.horizontal, 18)
        .padding(.vertical, 14)
    }
}

private struct MapCandidateList: View {
    let store: MapStore

    var body: some View {
        List {
            ForEach(store.candidates) { candidate in
                Button {
                    store.selectedCandidate = candidate
                } label: {
                    MapCandidateRow(candidate: candidate)
                }
                .buttonStyle(.plain)
                .listRowBackground(Color.clear)
                .accessibilityIdentifier("map.candidate.\(candidate.id)")
            }
        }
        .listStyle(.plain)
        .scrollContentBackground(.hidden)
        .overlay {
            if store.candidates.isEmpty {
                ContentUnavailableView(
                    "No Railway Candidates",
                    systemImage: "mappin.and.ellipse",
                    description: Text("Search the visible map area to discover stations from OpenStreetMap.")
                )
            }
        }
    }
}

private struct MapCandidateRow: View {
    let candidate: OverpassCandidate

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: candidate.importedStationId == nil ? "mappin.circle.fill" : "checkmark.circle.fill")
                .foregroundStyle(candidate.importedStationId == nil ? RailwayTheme.accent : .green)
            VStack(alignment: .leading, spacing: 3) {
                Text(candidate.name)
                    .foregroundStyle(.primary)
                if !details.isEmpty {
                    Text(details)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }
            }
            Spacer(minLength: 0)
            Image(systemName: "chevron.right")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.tertiary)
        }
        .contentShape(.rect)
    }

    private var details: String {
        [candidate.ref, candidate.mode, candidate.network]
            .compactMap { value in
                guard let value, !value.isEmpty else { return nil }
                return value
            }
            .joined(separator: " • ")
    }
}
