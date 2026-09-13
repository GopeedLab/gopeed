//
//  CGRect.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 10/05/22.
//

import Foundation

extension CGRect {
    public static func fromMap(map: [String: Double]) -> CGRect {
        return CGRect(x: map["x"]!, y: map["y"]!, width: map["width"]!, height: map["height"]!)
    }
    
    public func toMap () -> [String:Any?] {
        return [
            "x": minX,
            "y": minY,
            "width": width,
            "height": height
        ]
    }
}
