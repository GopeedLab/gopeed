//
//  pullToRefreshSettings.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 03/03/21.
//

import Foundation

public class PullToRefreshSettings: ISettings<PullToRefreshControl> {
    
    var enabled = true
    var color: String?
    var backgroundColor: String?
    var attributedTitle: [String: Any?]?
    
    override init(){
        super.init()
    }
    
    override func parse(settings: [String: Any?]) -> PullToRefreshSettings {
        let _ = super.parse(settings: settings)
        if let attributedTitle = settings["attributedTitle"] as? [String: Any?] {
            self.attributedTitle = attributedTitle
        }
        return self
    }
    
    override func getRealSettings(obj: PullToRefreshControl?) -> [String: Any?] {
        let realSettings: [String: Any?] = toMap()
        return realSettings
    }
}
