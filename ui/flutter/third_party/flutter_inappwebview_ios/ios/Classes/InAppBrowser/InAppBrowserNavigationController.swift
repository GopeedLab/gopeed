//
//  InAppBrowserNavigationController.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 14/02/21.
//

import Foundation

public class InAppBrowserNavigationController: UINavigationController {
    var tmpWindow: UIWindow?

    deinit {
        debugPrint("InAppBrowserNavigationController - dealloc")
        tmpWindow?.windowLevel = UIWindow.Level(rawValue: 0.0)
        tmpWindow = nil
        UIApplication.shared.delegate?.window??.makeKeyAndVisible()
    }
}
