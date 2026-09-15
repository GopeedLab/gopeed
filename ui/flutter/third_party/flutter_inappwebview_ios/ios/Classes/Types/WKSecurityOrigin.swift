//
//  WKSecurityOrigin.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 19/02/21.
//

import Foundation
import WebKit

@available(iOS 9.0, *)
extension WKSecurityOrigin {
    public func toMap () -> [String:Any?] {
        return [
            "host": host,
            "port": port,
            "protocol": self.protocol
        ]
    }
}
