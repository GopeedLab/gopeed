//
//  ActivityButton.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 26/10/22.
//

import Foundation
import SafariServices

@available(iOS 15.0, *)
extension SFSafariViewController.ActivityButton {
    public static func fromMap(map: [String:Any?]?) -> SFSafariViewController.ActivityButton? {
        guard let map = map else {
            return nil
        }
        if let templateImageMap = map["templateImage"] as? [String:Any?],
           let templateImage = UIImage.fromMap(map: templateImageMap),
           let extensionIdentifier = map["extensionIdentifier"] as? String {
            return .init(templateImage: templateImage, extensionIdentifier: extensionIdentifier)
        }
        return nil
    }
}
