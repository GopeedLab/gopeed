//
//  NSImage.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 23/05/23.
//

import Foundation
import FlutterMacOS

extension NSImage {
    public static func fromMap(map: [String:Any?]?) -> NSImage? {
        guard let map = map else {
            return nil
        }
        if let name = map["name"] as? String {
            return NSImage(named: name)
        }
        if #available(macOS 11.0, *), let systemName = map["systemName"] as? String {
            return NSImage(systemSymbolName: systemName, accessibilityDescription: nil)
        }
        if let data = map["data"] as? FlutterStandardTypedData {
            return NSImage(data: data.data)
        }
        return nil
    }
    
    func tint(color: NSColor) -> NSImage {
        return NSImage(size: size, flipped: false) { (rect) -> Bool in
            color.set()
            rect.fill()
            self.draw(in: rect, from: NSRect(origin: .zero, size: self.size), operation: .destinationIn, fraction: 1.0)
            return true
        }
    }
}
