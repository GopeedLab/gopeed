//
//  CGSize.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 28/10/22.
//

import Foundation

extension CGSize {
    public static func fromMap(map: [String: Double]) -> CGSize {
        return CGSize(width: map["width"]!, height: map["height"]!)
    }
    
    public func toMap () -> [String:Any?] {
        return [
            "width": width,
            "height": height
        ]
    }
}
