//
//  JsConfirmResponse.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 07/05/22.
//

import Foundation

public class JsConfirmResponse: NSObject {
    var message: String
    var confirmButtonTitle: String
    var cancelButtonTitle: String
    var handledByClient: Bool
    var action: Int?
    
    public init(message: String, confirmButtonTitle: String, cancelButtonTitle: String, handledByClient: Bool, action: Int? = nil) {
        self.message = message
        self.confirmButtonTitle = confirmButtonTitle
        self.cancelButtonTitle = cancelButtonTitle
        self.handledByClient = handledByClient
        self.action = action
    }
    
    public static func fromMap(map: [String:Any?]?) -> JsConfirmResponse? {
        guard let map = map else {
            return nil
        }
        let message = map["message"] as! String
        let confirmButtonTitle = map["confirmButtonTitle"] as! String
        let cancelButtonTitle = map["cancelButtonTitle"] as! String
        let handledByClient = map["handledByClient"] as! Bool
        let action = map["action"] as? Int
        return JsConfirmResponse(message: message, confirmButtonTitle: confirmButtonTitle, cancelButtonTitle: cancelButtonTitle, handledByClient: handledByClient, action: action)
    }
}
