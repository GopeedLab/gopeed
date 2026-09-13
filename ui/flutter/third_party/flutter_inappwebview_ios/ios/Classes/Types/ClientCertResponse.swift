//
//  ClientCertResponse.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 07/05/22.
//

import Foundation

public class ClientCertResponse: NSObject {
    var certificatePath: String
    var certificatePassword: String?
    var keyStoreType: String?
    var action: Int?
    
    public init(certificatePath: String, certificatePassword: String? = nil, keyStoreType: String? = nil, action: Int? = nil) {
        self.certificatePath = certificatePath
        self.certificatePassword = certificatePassword
        self.keyStoreType = keyStoreType
        self.action = action
    }
    
    public static func fromMap(map: [String:Any?]?) -> ClientCertResponse? {
        guard let map = map else {
            return nil
        }
        let certificatePath = map["certificatePath"] as! String
        let certificatePassword = map["certificatePassword"] as? String
        let keyStoreType = map["keyStoreType"] as? String
        let action = map["action"] as? Int
        return ClientCertResponse(certificatePath: certificatePath, certificatePassword: certificatePassword, keyStoreType: keyStoreType, action: action)
    }
}
