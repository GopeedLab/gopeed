//
//  WebResourceError.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 01/05/22.
//

import Foundation

public class WebResourceError: NSObject {
    var type: Int
    var errorDescription: String
    
    public init(type: Int, errorDescription: String) {
        self.type = type
        self.errorDescription = errorDescription
    }
    
    public func toMap () -> [String:Any?] {
        return [
            "type": type,
            "description": errorDescription
        ]
    }
}
