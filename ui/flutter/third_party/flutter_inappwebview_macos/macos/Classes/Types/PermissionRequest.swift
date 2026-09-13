//
//  PermissionRequest.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 21/04/22.
//

import Foundation
import WebKit

public class PermissionRequest: NSObject {
    var origin: String
    var resources: [StringOrInt]
    var frame: WKFrameInfo
    
    public init(origin: String, resources: [StringOrInt], frame: WKFrameInfo) {
        self.origin = origin
        self.resources = resources
        self.frame = frame
    }
    
    public func toMap () -> [String:Any?] {
        return [
            "origin": origin,
            "resources": resources,
            "frame": frame.toMap()
        ]
    }
}
