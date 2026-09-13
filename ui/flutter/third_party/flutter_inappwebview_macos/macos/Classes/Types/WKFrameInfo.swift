//
//  WKFrameInfo.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 19/02/21.
//

import Foundation
import WebKit

extension WKFrameInfo {
    
    public func toMap () -> [String:Any?] {
        var securityOrigin: [String:Any?]? = nil
        securityOrigin = self.securityOrigin.toMap()
        // fix: self.request throws EXC_BREAKPOINT when coming from WKNavigationAction.sourceFrame
        let request: URLRequest? = self.value(forKey: "request") as? URLRequest
        return [
            "isMainFrame": isMainFrame,
            "request": request?.toMap(),
            "securityOrigin": securityOrigin
        ]
    }
}
