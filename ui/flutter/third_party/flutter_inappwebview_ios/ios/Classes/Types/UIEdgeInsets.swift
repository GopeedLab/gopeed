//
//  UIEdgeInsets.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 11/05/22.
//

import Foundation

extension UIEdgeInsets {
    public static func fromMap(map: [String: Double]) -> UIEdgeInsets {
        return UIEdgeInsets.init(top: map["top"]!, left: map["left"]!, bottom: map["bottom"]!, right: map["right"]!)
    }
    
    public func toMap () -> [String:Any?] {
        return [
            "top": top,
            "right": self.right,
            "bottom": bottom,
            "left": self.left
        ]
    }
}
