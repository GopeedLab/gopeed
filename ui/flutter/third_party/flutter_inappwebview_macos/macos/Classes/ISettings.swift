//
//  Options.swift
//  flutter_inappwebview
//
//  Created by Lorenzo on 26/09/18.
//

import Foundation

@objcMembers
public class ISettings<T>: NSObject {
    
    override init(){
        super.init()
    }
    
    func parse(settings: [String: Any?]) -> ISettings<T> {
        for (key, value) in settings {
            if !(value is NSNull), value != nil {
                if self.responds(to: Selector(key)) {
                    self.setValue(value, forKey: key)
                } else if self.responds(to: Selector("_" + key)) {
                    self.setValue(value, forKey: "_" + key)
                }
            }
        }
        return self
    }
    
    func toMap() -> [String: Any?] {
        var settings: [String: Any?] = [:]
        let mirrored_object = Mirror(reflecting: self)
        for (_, attr) in mirrored_object.children.enumerated() {
            if let property_name = attr.label as String? {
                settings[property_name] = attr.value
            }
        }
        return settings
    }
    
    func getRealSettings(obj: T?) -> [String: Any?] {
        let realSettings: [String: Any?] = toMap()
        return realSettings
    }
}
