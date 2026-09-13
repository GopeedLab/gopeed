//
//  ServerTrustAuthResponse.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 07/05/22.
//

import Foundation

public class ServerTrustAuthResponse: NSObject {
    var action: Int?
    
    public init(action: Int? = nil) {
        self.action = action
    }
    
    public static func fromMap(map: [String:Any?]?) -> ServerTrustAuthResponse? {
        guard let map = map else {
            return nil
        }
        let action = map["action"] as? Int
        return ServerTrustAuthResponse(action: action)
    }
}
