//
//  JsPromptResponse.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 07/05/22.
//

import Foundation

public class JsPromptResponse: NSObject {
    var message: String
    var defaultValue: String
    var confirmButtonTitle: String
    var cancelButtonTitle: String
    var handledByClient: Bool
    var value: String?
    var action: Int?
    
    public init(message: String, defaultValue: String, confirmButtonTitle: String, cancelButtonTitle: String, handledByClient: Bool, value: String? = nil, action: Int? = nil) {
        self.message = message
        self.defaultValue = defaultValue
        self.confirmButtonTitle = confirmButtonTitle
        self.cancelButtonTitle = cancelButtonTitle
        self.handledByClient = handledByClient
        self.value = value
        self.action = action
    }
    
    public static func fromMap(map: [String:Any?]?) -> JsPromptResponse? {
        guard let map = map else {
            return nil
        }
        let message = map["message"] as! String
        let defaultValue = map["defaultValue"] as! String
        let confirmButtonTitle = map["confirmButtonTitle"] as! String
        let cancelButtonTitle = map["cancelButtonTitle"] as! String
        let handledByClient = map["handledByClient"] as! Bool
        let value = map["value"] as? String
        let action = map["action"] as? Int
        return JsPromptResponse(message: message, defaultValue: defaultValue, confirmButtonTitle: confirmButtonTitle, cancelButtonTitle: cancelButtonTitle,
                                handledByClient: handledByClient, value: value, action: action)
    }
}
