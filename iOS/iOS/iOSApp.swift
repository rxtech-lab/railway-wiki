//
//  iOSApp.swift
//  iOS
//
//  Created by Qiwei Li on 7/11/26.
//

import SwiftUI

@main
struct RailwayWikiApp: App {
    private let bootstrap: AppBootstrap

    init() {
        do {
            let configuration = try AppConfiguration.load()
            MapNetworkConfigurator.configure(configuration: configuration)
            bootstrap = .ready(AppDependencies(configuration: configuration))
        } catch {
            bootstrap = .failed(error.localizedDescription)
        }
    }

    var body: some Scene {
        WindowGroup {
            ContentView(bootstrap: bootstrap)
        }
    }
}

enum AppBootstrap {
    case ready(AppDependencies)
    case failed(String)
}
