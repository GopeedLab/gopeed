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
        var counts = UInt32()
        let properties = class_copyPropertyList(object_getClass(self), &counts)
        for i in 0..<counts {
            if let property = properties?.advanced(by: Int(i)).pointee {
                let cName = property_getName(property)
                let name = String(cString: cName)
                let key = !name.hasPrefix("_") ? name : String(name.suffix(from: name.index(name.startIndex, offsetBy: 1)))
                settings[key] = self.value(forKey: key)
            }
        }
        free(properties)
        return settings
    }
    
    func getRealSettings(obj: T?) -> [String: Any?] {
        let realSettings: [String: Any?] = toMap()
        return realSettings
    }
}
