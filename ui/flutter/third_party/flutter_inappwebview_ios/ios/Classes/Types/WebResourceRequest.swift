//
//  WebResourceRequest.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 01/05/22.
//

import Foundation
import WebKit

public class WebResourceRequest: NSObject {
    var url: URL
    var headers: [AnyHashable:Any]?
    var isRedirect = false
    var hasGesture = false
    var isForMainFrame = true
    var method = "GET"
    
    public init(url: URL, headers: [AnyHashable:Any]?) {
        self.url = url
        self.headers = headers
    }
    
    public init(url: URL, headers: [AnyHashable:Any]?, isForMainFrame: Bool) {
        self.url = url
        self.headers = headers
        self.isForMainFrame = isForMainFrame
    }
    
    public init(fromURLRequest: URLRequest) {
        self.url = fromURLRequest.url ?? URL(string: "about:blank")!
        self.headers = fromURLRequest.allHTTPHeaderFields
        self.method = fromURLRequest.httpMethod ?? "GET"
    }
    
    public init(fromWKNavigationResponse: WKNavigationResponse) {
        let response = fromWKNavigationResponse.response as? HTTPURLResponse
        self.url = response?.url ?? URL(string: "about:blank")!
        self.headers = response?.allHeaderFields
        self.isForMainFrame = fromWKNavigationResponse.isForMainFrame
    }
    
    public func toMap () -> [String:Any?] {
        return [
            "url": url.absoluteString,
            "headers": headers,
            "isRedirect": isRedirect,
            "hasGesture": hasGesture,
            "isForMainFrame": isForMainFrame,
            "method": method
        ]
    }
}
