//
//  JsAlertResponse.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 06/05/22.
//

import Foundation

public class JsAlertResponse: NSObject {
    var message: String
    var confirmButtonTitle: String
    var handledByClient: Bool
    var action: Int?
    
    public init(message: String, confirmButtonTitle: String, handledByClient: Bool, action: Int? = nil) {
        self.message = message
        self.confirmButtonTitle = confirmButtonTitle
        self.handledByClient = handledByClient
        self.action = action
    }
    
    public static func fromMap(map: [String:Any?]?) -> JsAlertResponse? {
        guard let map = map else {
            return nil
        }
        let message = map["message"] as! String
        let confirmButtonTitle = map["confirmButtonTitle"] as! String
        let handledByClient = map["handledByClient"] as! Bool
        let action = map["action"] as? Int
        return JsAlertResponse(message: message, confirmButtonTitle: confirmButtonTitle, handledByClient: handledByClient, action: action)
    }
}
