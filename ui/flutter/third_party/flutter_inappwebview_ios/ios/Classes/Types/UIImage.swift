//
//  UIImage.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 26/10/22.
//

import Foundation
import Flutter

extension UIImage {
    public static func fromMap(map: [String:Any?]?) -> UIImage? {
        guard let map = map else {
            return nil
        }
        if let name = map["name"] as? String {
            return UIImage(named: name)
        }
        if #available(iOS 13.0, *), let systemName = map["systemName"] as? String {
            return UIImage(systemName: systemName)
        }
        if let data = map["data"] as? FlutterStandardTypedData {
            return UIImage(data: data.data)
        }
        return nil
    }
}
