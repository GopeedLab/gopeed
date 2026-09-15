//
//  URLCredential.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 19/02/21.
//

import Foundation

extension URLCredential {
    public func toMap () -> [String:Any?] {
        var x509Certificates: [Data] = []
        // certificates could be nil!!!
        if certificates != nil {
            for certificate in certificates {
                x509Certificates.append((certificate as! SecCertificate).data)
            }
        }
        return [
            "password": password,
            "username": user,
            "certificates": x509Certificates,
            "persistence": persistence.rawValue
        ]
    }
}
