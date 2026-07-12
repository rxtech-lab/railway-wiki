import Foundation
import JSONSchemaForm
import SwiftUI
import UniformTypeIdentifiers

enum FormWidgetRegistry {
    static func widgets(api: APIClient) -> [String: JSONSchemaFormWidget] {
        [
            "relation": { AnyView(RelationFormWidget(context: $0, api: api)) },
            "coordinate": { AnyView(CoordinateFormWidget(context: $0)) },
            "geojson": { AnyView(JSONFormWidget(context: $0)) },
            "media-upload": { AnyView(MediaUploadFormWidget(context: $0, api: api)) }
        ]
    }
}

private struct RelationFormWidget: View {
    let context: JSONSchemaFormWidgetContext
    let api: APIClient
    @State private var options: [ResourceRecord] = []

    var body: some View {
        Picker(context.propertyName?.humanized ?? "Related Record", selection: stringBinding) {
            Text("None").tag("")
            ForEach(options) { option in
                Text(option.title).tag(option.stableID ?? "")
            }
        }
        .accessibilityIdentifier(accessibilityID)
        .task {
            guard let field = context.propertyName,
                  let resource = ResourceDefinition.relationResource(for: field)
            else { return }
            options = (try? await api.list(resource).items) ?? []
        }
    }

    private var stringBinding: Binding<String> {
        Binding {
            context.formData.wrappedValue.string ?? ""
        } set: { value in
            context.formData.wrappedValue = value.isEmpty ? .null : .string(value)
        }
    }

    private var accessibilityID: String {
        (context.uiSchema?["ui:options"] as? [String: Any])?["accessibility_id"] as? String ?? context.id
    }
}

private struct CoordinateFormWidget: View {
    let context: JSONSchemaFormWidgetContext

    var body: some View {
        TextField(context.propertyName?.humanized ?? "Coordinate", value: numberBinding, format: .number)
            .keyboardType(.numbersAndPunctuation)
            .accessibilityIdentifier(accessibilityID)
    }

    private var numberBinding: Binding<Double> {
        Binding { context.formData.wrappedValue.number ?? 0 } set: { context.formData.wrappedValue = .number($0) }
    }

    private var accessibilityID: String {
        (context.uiSchema?["ui:options"] as? [String: Any])?["accessibility_id"] as? String ?? context.id
    }
}

private struct MediaUploadFormWidget: View {
    let context: JSONSchemaFormWidgetContext
    let api: APIClient
    @State private var presentsImporter = false
    @State private var isUploading = false
    @State private var errorMessage: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            TextField("Media URL", text: urlBinding)
                .textInputAutocapitalization(.never)
            HStack {
                Button("Choose File", systemImage: "arrow.up.doc") { presentsImporter = true }
                    .disabled(isUploading)
                if isUploading { ProgressView().controlSize(.small) }
            }
            if let errorMessage {
                Text(errorMessage).font(.caption).foregroundStyle(.red)
            }
        }
        .accessibilityIdentifier(context.id)
        .fileImporter(isPresented: $presentsImporter, allowedContentTypes: [.data]) { result in
            Task { await importFile(result) }
        }
    }

    private var urlBinding: Binding<String> {
        Binding {
            context.formData.wrappedValue.string ?? ""
        } set: {
            context.formData.wrappedValue = .string($0)
        }
    }

    private func importFile(_ result: Result<URL, Error>) async {
        isUploading = true
        defer { isUploading = false }
        do {
            let url = try result.get()
            let scoped = url.startAccessingSecurityScopedResource()
            defer { if scoped { url.stopAccessingSecurityScopedResource() } }
            let type = try url.resourceValues(forKeys: [.contentTypeKey]).contentType
            let contentType = type?.preferredMIMEType ?? "application/octet-stream"
            let uploaded = try await api.uploadMedia(
                fileURL: url,
                contentType: contentType,
                mediaType: mediaType(for: type)
            )
            context.formData.wrappedValue = .string(uploaded.absoluteString)
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func mediaType(for type: UTType?) -> String? {
        if type?.conforms(to: .image) == true { return "image" }
        if type?.conforms(to: .movie) == true { return "video" }
        if type?.conforms(to: .audio) == true { return "audio" }
        return nil
    }
}

private struct JSONFormWidget: View {
    let context: JSONSchemaFormWidgetContext
    @State private var text: String

    init(context: JSONSchemaFormWidgetContext) {
        self.context = context
        let value = JSONValue(formData: context.formData.wrappedValue)
        let data = try? JSONEncoder().encode(value)
        _text = State(initialValue: data.flatMap { String(data: $0, encoding: .utf8) } ?? "{}")
    }

    var body: some View {
        VStack(alignment: .leading) {
            Text(context.propertyName?.humanized ?? "GeoJSON").font(.caption)
            TextEditor(text: $text)
                .font(.system(.body, design: .monospaced))
                .frame(minHeight: 100)
                .onChange(of: text) { _, value in
                    guard let data = value.data(using: .utf8),
                          let decoded = try? JSONDecoder().decode(JSONValue.self, from: data)
                    else { return }
                    context.formData.wrappedValue = decoded.formData
                }
        }
        .accessibilityIdentifier(context.id)
    }
}
