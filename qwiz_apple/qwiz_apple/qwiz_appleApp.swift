//
//  qwiz_appleApp.swift
//  qwiz_apple
//
//  Created by Malkorin Play on 10/26/23.
//

import SwiftUI

@main
struct qwiz_appleApp: App {
    let persistenceController = PersistenceController.shared

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environment(\.managedObjectContext, persistenceController.container.viewContext)
        }
    }
}
