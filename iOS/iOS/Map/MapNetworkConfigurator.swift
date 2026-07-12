import Foundation
import MapLibre

enum MapNetworkConfigurator {
    @MainActor
    static func configure(configuration: AppConfiguration) {
        MapAPIKeyURLProtocol.configure(host: configuration.osmStackBaseURL.host, apiKey: configuration.osmMapsAPIKey)
        let session = URLSessionConfiguration.default
        session.protocolClasses = [MapAPIKeyURLProtocol.self] + (session.protocolClasses ?? [])
        MLNNetworkConfiguration.sharedManager.sessionConfiguration = session
    }
}

final class MapAPIKeyURLProtocol: URLProtocol {
    private static let lock = NSLock()
    private static var targetHost: String?
    private static var apiKey: String?
    private static let handledKey = "RailwayWikiMapAPIKeyHandled"
    private var dataTask: URLSessionDataTask?

    static func configure(host: String?, apiKey: String) {
        lock.lock()
        targetHost = host?.lowercased()
        self.apiKey = apiKey
        lock.unlock()
    }

    override class func canInit(with request: URLRequest) -> Bool {
        guard URLProtocol.property(forKey: handledKey, in: request) == nil else { return false }
        lock.lock()
        let host = targetHost
        lock.unlock()
        return request.url?.host?.lowercased() == host
    }

    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }

    override func startLoading() {
        guard let request = (request as NSURLRequest).mutableCopy() as? NSMutableURLRequest else {
            client?.urlProtocol(self, didFailWithError: URLError(.badURL))
            return
        }
        Self.lock.lock()
        let key = Self.apiKey
        Self.lock.unlock()
        URLProtocol.setProperty(true, forKey: Self.handledKey, in: request)
        request.setValue(key, forHTTPHeaderField: "X-API-Key")
        dataTask = URLSession(configuration: .ephemeral).dataTask(
            with: request as URLRequest
        ) { [weak self] data, response, error in
            guard let self else { return }
            if let error {
                client?.urlProtocol(self, didFailWithError: error)
                return
            }
            if let response { client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .allowed) }
            if let data { client?.urlProtocol(self, didLoad: data) }
            client?.urlProtocolDidFinishLoading(self)
        }
        dataTask?.resume()
    }

    override func stopLoading() {
        dataTask?.cancel()
        dataTask = nil
    }
}
