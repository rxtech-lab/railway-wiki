import Foundation
import SwiftUI
import UIKit

struct ResourceDetailView: View {
    @Environment(AppDependencies.self) private var dependencies
    let resource: ResourceDefinition
    let onDeleted: () -> Void
    let header: AnyView?
    let showsActions: Bool
    @State private var record: ResourceRecord
    @State private var presentsEdit = false
    @State private var confirmsDelete = false
    @State private var errorMessage: String?

    init(
        resource: ResourceDefinition,
        record: ResourceRecord,
        header: AnyView? = nil,
        showsActions: Bool = true,
        onDeleted: @escaping () -> Void = {}
    ) {
        self.resource = resource
        self.onDeleted = onDeleted
        self.header = header
        self.showsActions = showsActions
        _record = State(initialValue: record)
    }

    var body: some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 12) {
                if let header { header }
                if let id = record.stableID {
                    IDCard(id: id)
                }
                ForEach(record.values.keys.sorted().filter { $0 != "id" }, id: \.self) { key in
                    valueRow(key: key, value: record.values[key] ?? .null)
                }
            }
            .padding()
        }
        .navigationTitle(record.title)
        .toolbar {
            if showsActions {
                ToolbarItem(placement: .primaryAction) {
                    Button("Edit", systemImage: "pencil") { presentsEdit = true }
                        .accessibilityIdentifier("record.edit")
                }
                ToolbarItem(placement: .secondaryAction) {
                    Menu {
                        Button("Copy ID", systemImage: "doc.on.doc") { UIPasteboard.general.string = record.stableID }
                        Button("Delete", systemImage: "trash", role: .destructive) { confirmsDelete = true }
                    } label: {
                        Label("More", systemImage: "ellipsis.circle")
                    }
                }
            }
        }
        .sheet(isPresented: $presentsEdit) {
            SchemaFormSheet(resource: resource, record: record, onSaved: { record = $0 })
        }
        .confirmationDialog("Delete \(record.title)?", isPresented: $confirmsDelete) {
            Button("Delete", role: .destructive) { Task { await deleteRecord() } }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("This action cannot be undone.")
        }
        .alert("Action Failed", isPresented: Binding(get: { errorMessage != nil }, set: { if !$0 { errorMessage = nil } })) {
            Button("OK") { errorMessage = nil }
        } message: { Text(errorMessage ?? "") }
        .accessibilityIdentifier("resource.\(resource.id).detail")
    }

    @ViewBuilder
    private func valueRow(key: String, value: JSONValue) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(key.humanized).font(.caption).foregroundStyle(.secondary)
            if let id = value.stringValue, let related = ResourceDefinition.relationResource(for: key) {
                RelatedRecordLabel(resource: related, id: id)
            } else {
                Text(value.displayValue).textSelection(.enabled)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .railwayGlassCard()
    }

    private func deleteRecord() async {
        guard let id = record.stableID else { return }
        do {
            try await dependencies.api.delete(resource, id: id)
            onDeleted()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

private struct IDCard: View {
    let id: String
    var body: some View {
        HStack {
            VStack(alignment: .leading, spacing: 4) {
                Text("Identifier").font(.caption).foregroundStyle(.secondary)
                Text(id).font(.system(.body, design: .monospaced)).textSelection(.enabled)
            }
            Spacer()
            Button("Copy", systemImage: "doc.on.doc") { UIPasteboard.general.string = id }.labelStyle(.iconOnly)
        }
        .railwayGlassCard()
    }
}

private struct RelatedRecordLabel: View {
    @Environment(AppDependencies.self) private var dependencies
    let resource: ResourceDefinition
    let id: String
    @State private var title: String?

    var body: some View {
        HStack {
            Text(title ?? id).textSelection(.enabled)
            if title == nil { ProgressView().controlSize(.small) }
            Spacer()
            Button("Copy ID", systemImage: "doc.on.doc") { UIPasteboard.general.string = id }.labelStyle(.iconOnly)
        }
        .task {
            title = try? await dependencies.api.get(resource, id: id).title
        }
    }
}

extension String {
    var humanized: String {
        unicodeScalars.reduce(into: "") { result, scalar in
            if CharacterSet.uppercaseLetters.contains(scalar), !result.isEmpty { result.append(" ") }
            result.append(String(scalar))
        }.replacingOccurrences(of: "Id", with: "ID").capitalized
    }
}
