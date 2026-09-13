//
//  Size.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 26/03/21.
//

import Foundation

public class Size2D: NSObject {
    var width: Double
    var height: Double
    
    public init(width: Double, height: Double) {
        self.width = width
        self.height = height
    }
    
    public static func fromMap(map: [String:Any?]?) -> Size2D? {
        guard let map = map else {
            return nil
        }
        return Size2D(
            width: map["width"] as? Double ?? -1.0,
            height: map["height"] as? Double ?? -1.0
        )
    }
    
    public func toMap() -> [String:Any?] {
        return [
            "width": width,
            "height": height
        ]
    }
}
