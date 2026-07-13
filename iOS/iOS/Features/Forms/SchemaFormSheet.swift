import Foundation
import JSONSchema
import JSONSchemaForm
import JSONSchemaValidator
import SwiftUI

struct SchemaFormSheet: View {
    @Environment(AppDependencies.self) private var dependencies
    @Environment(\.dismiss) private var dismiss
    let resource: ResourceDefinition
    let record: ResourceRecord?
    let customSave: (([String: JSONValue]) async throws -> ResourceRecord)?
    let onSaved: (ResourceRecord) -> Void

    @State private var schema: JSONSchema?
    @State private var fullSchema: JSONSchema?
    @State private var schemaJSON: String?
    @State private var uiSchema: [String: Any]?
    @State private var wireSchema: [String: JSONValue]?
    @State private var stages: [FormLayoutStage] = []
    @State private var selectedStageID: String?
    @State private var formData: FormData
    @State private var originalData: FormData
    @State private var controller = JSONSchemaFormController()
    @State private var isLoading = true
    @State private var isSaving = false
    @State private var errorMessage: String?
    @State private var confirmsClose = false
    /// Media uploaded from the create form, attached once the entity exists.
    @State private var pendingMedia: [ResourceRecord] = []
    /// The record created by a save whose photo attachments failed, so a
    /// retry updates it instead of creating a duplicate.
    @State private var createdRecord: ResourceRecord?

    init(
        resource: ResourceDefinition,
        record: ResourceRecord? = nil,
        customSave: (([String: JSONValue]) async throws -> ResourceRecord)? = nil,
        onSaved: @escaping (ResourceRecord) -> Void = { _ in }
    ) {
        self.resource = resource
        self.record = record
        self.customSave = customSave
        self.onSaved = onSaved
        let initial = FormData.object(properties: (record?.editableValues ?? [:]).mapValues(\.formData))
        _formData = State(initialValue: initial)
        _originalData = State(initialValue: initial)
    }

    var body: some View {
        NavigationStack {
            Form {
                if !stages.isEmpty {
                    Section { stagePicker }
                }
                if resource.specialization == .station, selectedStageID == "location" {
                    Section {
                        StationLocationTools(
                            configuration: dependencies.configuration,
                            formData: $formData
                        )
                    }
                }
                if let schema {
                    JSONSchemaForm(
                        schema: schema,
                        uiSchema: uiSchema,
                        formData: $formData,
                        schemaJSON: schemaJSON,
                        liveValidate: false,
                        showErrorList: true,
                        showSubmitButton: false,
                        widgets: FormWidgetRegistry.widgets(api: dependencies.api),
                        idPrefix: "form_\(resource.id)",
                        controller: controller
                    )
                    .id(selectedStageID)
                } else if let errorMessage {
                    ContentUnavailableView(
                        "Form Unavailable",
                        systemImage: "doc.badge.exclamationmark",
                        description: Text(errorMessage)
                    )
                }
                if let entityType = resource.mediaEntityType, schema != nil {
                    EntityPhotosSection(
                        api: dependencies.api,
                        entityType: entityType,
                        entityId: (record ?? createdRecord)?.stableID,
                        pendingMedia: $pendingMedia
                    )
                }
            }
            .formStyle(.grouped)
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle(record == nil ? "New \(resource.singular)" : "Edit \(resource.singular)")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Close", systemImage: "xmark") { requestClose() }
                        .accessibilityIdentifier("form.close")
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save", systemImage: "checkmark") { Task { await save() } }
                        .disabled(schema == nil || isSaving)
                        .accessibilityIdentifier("form.save")
                }
                if !stages.isEmpty {
                    ToolbarItem(placement: .primaryAction) {
                        Menu("Form Section", systemImage: "list.bullet") {
                            ForEach(stages) { stage in
                                Button(stage.title) { selectStage(stage.id) }
                            }
                        }
                    }
                }
            }
            .overlay { if isLoading || isSaving { ProgressView().railwayGlassCard() } }
        }
        .interactiveDismissDisabled(true)
        .presentationDragIndicator(.hidden)
        .confirmationDialog("Discard changes?", isPresented: $confirmsClose) {
            Button("Discard Changes", role: .destructive) {
                discardPendingMedia()
                dismiss()
            }
            Button("Keep Editing", role: .cancel) {}
        }
        .alert("Could Not Save", isPresented: errorBinding) {
            Button("OK") { errorMessage = nil }
        } message: { Text(errorMessage ?? "") }
        .task { await loadSchema() }
    }

    private var errorBinding: Binding<Bool> {
        Binding(
            get: { errorMessage != nil && schema != nil },
            set: { if !$0 { errorMessage = nil } }
        )
    }

    private func loadSchema() async {
        isLoading = true
        defer { isLoading = false }
        do {
            let action = record == nil ? "create" : "update"
            let wire = try await dependencies.api.schema(resource, action: action)
            wireSchema = wire
            fullSchema = try parseSchema(wire).schema
            stages = resource.specialization == .station ? FormLayoutStage.stages(from: wire) : []
            selectedStageID = stages.first?.id
            try applyDisplayedSchema()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func save() async {
        guard let fullSchema, case .object(let properties) = formData else { return }
        do {
            try JSONSchemaValidator.validate(formData.toDictionary(), schema: fullSchema)
        } catch {
            if stages.contains(where: { $0.id == "review" }) { selectStage("review") }
            errorMessage = "Review the highlighted fields before saving."
            return
        }
        guard controller.validate() else { return }
        let values = properties.mapValues(JSONValue.init)
        isSaving = true
        defer { isSaving = false }
        do {
            let saved: ResourceRecord
            if let customSave {
                saved = try await customSave(values)
            } else if let id = (record ?? createdRecord)?.stableID {
                saved = try await dependencies.api.update(resource, id: id, values: values)
            } else {
                saved = try await dependencies.api.create(resource, values: values)
            }
            if record == nil { createdRecord = saved }
            if let failed = await attachPendingMedia(to: saved) {
                originalData = formData
                onSaved(saved)
                errorMessage = failed
                return
            }
            originalData = formData
            onSaved(saved)
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    /// Attaches queued media to the saved entity. Returns an error message
    /// naming the failures (which stay queued for retry), or nil when done.
    private func attachPendingMedia(to saved: ResourceRecord) async -> String? {
        guard let entityType = resource.mediaEntityType,
              let entityId = saved.stableID,
              !pendingMedia.isEmpty
        else { return nil }
        var failures: [String] = []
        for (index, media) in pendingMedia.enumerated() {
            do {
                try await EntityPhotosSection.attach(
                    media: media,
                    entityType: entityType,
                    entityId: entityId,
                    sortOrder: index,
                    api: dependencies.api
                )
                pendingMedia.removeAll { $0.id == media.id }
            } catch {
                failures.append(media.title)
            }
        }
        guard !failures.isEmpty else { return nil }
        return "\(resource.singular) was saved, but these photos could not be attached: "
            + "\(failures.joined(separator: ", ")). Tap Save to retry."
    }

    private func requestClose() {
        if formData == originalData, pendingMedia.isEmpty { dismiss() } else { confirmsClose = true }
    }

    /// Deletes queued-but-unattached Media records when the form is discarded.
    private func discardPendingMedia() {
        let media = pendingMedia
        let api = dependencies.api
        pendingMedia = []
        Task {
            for item in media {
                if let id = item.stableID {
                    try? await api.delete(EntityPhotosSection.mediaResource, id: id)
                }
            }
        }
    }

    private var stagePicker: some View {
        Picker("Form section", selection: Binding(
            get: { selectedStageID ?? stages.first?.id ?? "" },
            set: selectStage
        )) {
            ForEach(stages) { stage in Text(stage.title).tag(stage.id) }
        }
        .pickerStyle(.segmented)
        .accessibilityIdentifier("form.station.stage")
    }

    private func selectStage(_ id: String) {
        selectedStageID = id
        do { try applyDisplayedSchema() } catch { errorMessage = error.localizedDescription }
    }

    private func applyDisplayedSchema() throws {
        guard let wireSchema else { return }
        let stage = stages.first { $0.id == selectedStageID }
        let parsed = try parseSchema(FormSchemaFilter.schema(wireSchema, for: stage))
        schema = parsed.schema
        schemaJSON = parsed.json
        uiSchema = parsed.uiSchema
        controller = JSONSchemaFormController()
    }

    private func parseSchema(_ wire: [String: JSONValue]) throws -> ParsedSchema {
        let data = try JSONEncoder().encode(JSONValue.object(wire))
        guard let json = String(data: data, encoding: .utf8),
              let root = try JSONSerialization.jsonObject(with: data) as? [String: Any]
        else { throw APIError.invalidResponse }
        return try ParsedSchema(
            schema: JSONSchema(jsonString: json),
            json: json,
            uiSchema: root["x-ui-schema"] as? [String: Any]
        )
    }
}

private struct ParsedSchema {
    let schema: JSONSchema
    let json: String
    let uiSchema: [String: Any]?
}
