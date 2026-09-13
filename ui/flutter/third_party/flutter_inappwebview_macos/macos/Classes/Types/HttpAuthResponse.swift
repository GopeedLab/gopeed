//
//  HttpAuthResponse.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 07/05/22.
//

import Foundation

public class HttpAuthResponse: NSObject {
    var username: String
    var password: String
    var permanentPersistence: Bool
    var action: Int?
    
    public init(username: String, password: String, permanentPersistence: Bool, action: Int? = nil) {
        self.username = username
        self.password = password
        self.permanentPersistence = permanentPersistence
        self.action = action
    }
    
    public static func fromMap(map: [String:Any?]?) -> HttpAuthResponse? {
        guard let map = map else {
            return nil
        }
        let username = map["username"] as! String
        let password = map["password"] as! String
        let permanentPersistence = map["permanentPersistence"] as! Bool
        let action = map["action"] as? Int
        return HttpAuthResponse(username: username, password: password, permanentPersistence: permanentPersistence, action: action)
    }
}
