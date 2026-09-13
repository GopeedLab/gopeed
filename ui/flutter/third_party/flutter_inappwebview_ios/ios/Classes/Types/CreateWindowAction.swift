//
//  CreateWindowAction.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 07/05/22.
//

import Foundation
import WebKit

public class CreateWindowAction: NSObject {
    var navigationAction: WKNavigationAction
    var windowId: Int64
    var windowFeatures: WKWindowFeatures
    var isDialog: Bool?
    
    public init(navigationAction: WKNavigationAction, windowId: Int64, windowFeatures: WKWindowFeatures, isDialog: Bool? = nil) {
        self.navigationAction = navigationAction
        self.windowId = windowId
        self.windowFeatures = windowFeatures
        self.isDialog = isDialog
    }
    
    public func toMap () -> [String:Any?] {
        var map = navigationAction.toMap()
        map["windowId"] = windowId
        map["windowFeatures"] = windowFeatures.toMap()
        map["isDialog"] = isDialog
        return map
    }
}
