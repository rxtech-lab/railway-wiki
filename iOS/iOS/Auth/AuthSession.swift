import Foundation
import Observation
import RxAuthSwift

@MainActor
@Observable
final class AuthSession {
    static let keychainService = "app.rxlab.railwaywiki.rxauth"

    let manager: OAuthManager
    private(set) var hasLostAdminAccess = false
    private let tokenStorage: KeychainTokenStorage
    private let testToken: String?

    init(configuration: AppConfiguration) {
        let storage = KeychainTokenStorage(serviceName: Self.keychainService)
        tokenStorage = storage
        testToken = configuration.testAuthToken
        let authConfiguration = RxAuthConfiguration(
            issuer: configuration.authIssuerURL.absoluteString.trimmingCharacters(in: CharacterSet(charactersIn: "/")),
            clientID: configuration.authClientID,
            redirectURI: configuration.authRedirectURI,
            scopes: configuration.authScopes,
            keychainServiceName: Self.keychainService
        )
        manager = OAuthManager(configuration: authConfiguration, tokenStorage: storage)
    }

    var isTestAuthenticated: Bool { testToken != nil }
    var accessToken: String? { testToken ?? tokenStorage.getAccessToken() }

    func restore() async {
        guard testToken == nil else { return }
        await manager.checkExistingAuth()
    }

    func refreshedAccessToken() async -> String? {
        if let testToken { return testToken }
        try? await manager.refreshTokenIfNeeded()
        return tokenStorage.getAccessToken()
    }

    func markAdminAccessDenied() {
        hasLostAdminAccess = true
    }

    func restoreAdminAccess() {
        hasLostAdminAccess = false
    }

    func signOut() async {
        hasLostAdminAccess = false
        guard testToken == nil else { return }
        await manager.logout()
    }
}
