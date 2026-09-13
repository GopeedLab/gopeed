//
//  FindInteractionSettings.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 07/10/22.
//

import Foundation

public class FindInteractionSettings: ISettings<FindInteractionController> {
    
    override init(){
        super.init()
    }
    
    override func parse(settings: [String: Any?]) -> FindInteractionSettings {
        let _ = super.parse(settings: settings)
        return self
    }
    
    override func getRealSettings(obj: FindInteractionController?) -> [String: Any?] {
        let realSettings: [String: Any?] = toMap()
        return realSettings
    }
}
