//
//  WKContentWorld.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 19/02/21.
//

import Foundation
import WebKit

@available(iOS 14.0, *)
extension WKContentWorld {
    // Workaround to create stored properties in an extension:
    // https://valv0.medium.com/computed-properties-and-extensions-a-pure-swift-approach-64733768112c
    
    private static var _windowId = [String: Int64?]()

    var windowId: Int64? {
        get {
            let tmpAddress = String(format: "%p", unsafeBitCast(self, to: Int.self))
            return WKContentWorld._windowId[tmpAddress] ?? nil
        }
        set(newValue) {
            let tmpAddress = String(format: "%p", unsafeBitCast(self, to: Int.self))
            WKContentWorld._windowId[tmpAddress] = newValue
        }
    }
    
    public static func fromMap(map: [String:Any?]?, windowId: Int64?) -> WKContentWorld? {
        guard let map = map else {
            return nil
        }
        var name = map["name"] as! String
        name = windowId != nil && name != "page" ?
            WKUserContentController.WINDOW_ID_PREFIX + String(windowId!) + "-" + name :
            name
        let contentWorld = Util.getContentWorld(name: name)
        contentWorld.windowId = name != "page" ? windowId : nil
        return contentWorld
    }
}
