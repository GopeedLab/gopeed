//
//  WKNavigationAction.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 19/02/21.
//

import Foundation
import WebKit

extension WKNavigationAction {
    public func toMap () -> [String:Any?] {
        var shouldPerformDownload: Bool? = nil
        if #available(iOS 14.5, *) {
            shouldPerformDownload = self.shouldPerformDownload
        }
        return [
            "request": request.toMap(),
            "isForMainFrame": targetFrame?.isMainFrame ?? false,
            "hasGesture": nil,
            "isRedirect": nil,
            "navigationType": navigationType.rawValue,
            "sourceFrame": sourceFrame.toMap(),
            "targetFrame": targetFrame?.toMap(),
            "shouldPerformDownload": shouldPerformDownload
        ]
    }
}
