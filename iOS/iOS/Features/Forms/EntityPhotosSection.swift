import SwiftUI
import UniformTypeIdentifiers

/// Media attachments embedded in an entity form. In edit mode it lists the
/// entity's existing attachments and attaches uploads immediately; in create
/// mode uploads are queued in `pendingMedia` and attached by the form's save
/// once the entity id exists.
struct EntityPhotosSection: View {
    let api: APIClient
    let entityType: String
    let entityId: String?
    @Binding var pendingMedia: [ResourceRecord]

    @State private var attachments: [AttachedMedia] = []
    @State private var isLoading = false
    @State private var isUploading = false
    @State private var presentsImporter = false
    @State private var errorMessage: String?

    static let mediaResource = ResourceDefinition.all.first { $0.id == "media" }!
    static let attachmentsResource = ResourceDefinition.all.first { $0.id == "media-attachments" }!

    struct AttachedMedia: Identifiable {
        let attachment: ResourceRecord
        let media: ResourceRecord?
        var id: String { attachment.id }
    }

    var body: some View {
        Section("Photos") {
            if isLoading {
                ProgressView().frame(maxWidth: .infinity)
            }
            ForEach(attachments) { item in
                row(media: item.media, title: item.media?.title ?? "Media") {
                    Task { await removeAttachment(item) }
                }
            }
            ForEach(pendingMedia) { media in
                row(media: media, title: media.title, badge: "Attached on save") {
                    Task { await removePending(media) }
                }
            }
            HStack {
                Button("Add Photo", systemImage: "photo.badge.plus") { presentsImporter = true }
                    .disabled(isUploading)
                    .accessibilityIdentifier("form.photos.add")
                if isUploading { ProgressView().controlSize(.small) }
            }
            if let errorMessage {
                Text(errorMessage).font(.caption).foregroundStyle(.red)
            }
        }
        .accessibilityIdentifier("form.photos")
        .fileImporter(
            isPresented: $presentsImporter,
            allowedContentTypes: [.image, .movie, .audio]
        ) { result in
            Task { await importFile(result) }
        }
        .task { await refresh() }
    }

    /// Creates the MediaAttachment join record linking a Media record to an entity.
    static func attach(
        media: ResourceRecord,
        entityType: String,
        entityId: String,
        sortOrder: Int,
        api: APIClient
    ) async throws {
        guard let mediaId = media.stableID else { throw APIError.invalidResponse }
        _ = try await api.create(attachmentsResource, values: [
            "mediaId": .string(mediaId),
            "entityType": .string(entityType),
            "entityId": .string(entityId),
            "sortOrder": .number(Double(sortOrder))
        ])
    }

    @ViewBuilder
    private func row(media: ResourceRecord?, title: String, badge: String? = nil, onDelete: @escaping () -> Void) -> some View {
        HStack(spacing: 12) {
            thumbnail(for: media)
            VStack(alignment: .leading, spacing: 2) {
                Text(title).lineLimit(1)
                if let badge {
                    Text(badge).font(.caption).foregroundStyle(.secondary)
                }
            }
            Spacer()
            Button(role: .destructive) {
                onDelete()
            } label: {
                Image(systemName: "trash")
            }
            .buttonStyle(.borderless)
        }
    }

    @ViewBuilder
    private func thumbnail(for media: ResourceRecord?) -> some View {
        let url = media?.values["url"]?.stringValue.flatMap(URL.init(string:))
        let isImage = media?.values["mediaType"]?.stringValue == "image"
        if isImage, let url {
            AsyncImage(url: url) { image in
                image.resizable().scaledToFill()
            } placeholder: {
                ProgressView().controlSize(.small)
            }
            .frame(width: 44, height: 44)
            .clipShape(RoundedRectangle(cornerRadius: 8))
        } else {
            Image(systemName: "doc.richtext")
                .frame(width: 44, height: 44)
                .foregroundStyle(.secondary)
        }
    }

    private func refresh() async {
        guard let entityId else { return }
        isLoading = true
        defer { isLoading = false }
        do {
            let page = try await api.list(
                Self.attachmentsResource,
                filters: ["entityType": entityType, "entityId": entityId]
            )
            var rows: [AttachedMedia] = []
            for attachment in page.items {
                var media: ResourceRecord?
                if let mediaId = attachment.values["mediaId"]?.stringValue {
                    media = try? await api.get(Self.mediaResource, id: mediaId)
                }
                rows.append(.init(attachment: attachment, media: media))
            }
            attachments = rows
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
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
            let mediaType = mediaType(for: type)
            let uploaded = try await api.uploadMedia(
                fileURL: url,
                contentType: contentType,
                mediaType: mediaType
            )
            let media = try await api.create(Self.mediaResource, values: [
                "mediaType": .string(mediaType),
                "url": .string(uploaded.absoluteString),
                "title": .string(url.lastPathComponent)
            ])
            if let entityId {
                try await Self.attach(
                    media: media,
                    entityType: entityType,
                    entityId: entityId,
                    sortOrder: attachments.count,
                    api: api
                )
                await refresh()
            } else {
                pendingMedia.append(media)
            }
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    /// Detaches the media from the entity. The Media record is kept — it may
    /// be attached to other entities.
    private func removeAttachment(_ item: AttachedMedia) async {
        guard let id = item.attachment.stableID else { return }
        do {
            try await api.delete(Self.attachmentsResource, id: id)
            await refresh()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    /// Drops a queued upload; the just-created Media record is deleted so a
    /// discarded form leaves no orphans behind.
    private func removePending(_ media: ResourceRecord) async {
        if let id = media.stableID {
            try? await api.delete(Self.mediaResource, id: id)
        }
        pendingMedia.removeAll { $0.id == media.id }
    }

    private func mediaType(for type: UTType?) -> String {
        if type?.conforms(to: .movie) == true { return "video" }
        if type?.conforms(to: .audio) == true { return "audio" }
        return "image"
    }
}
