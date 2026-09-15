//
//  InAppBrowserMenuItem.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 21/05/23.
//

import Foundation

public class InAppBrowserMenuItem: NSObject {
    var id: Int64
    var title: String
    var order: Int64?
    var icon: UIImage?
    var iconColor: UIColor?
    var showAsAction = false
    
    public init(id: Int64, title: String, order: Int64?, icon: UIImage?, iconColor: UIColor?, showAsAction: Bool) {
        self.id = id
        self.title = title
        self.order = order
        self.icon = icon
        self.iconColor = iconColor
        self.showAsAction = showAsAction
        if #available(iOS 13.0, *), let icon = icon, let iconColor = iconColor {
            icon.withTintColor(iconColor, renderingMode: .alwaysOriginal)
        }
    }
    
    public static func fromMap(map: [String:Any?]?) -> InAppBrowserMenuItem? {
        guard let map = map else {
            return nil
        }
        let id = map["id"] as! Int64
        let title = map["title"] as! String
        let order = map["order"] as? Int64
        var icon = UIImage.fromMap(map: map["icon"] as? [String : Any?])
        if let data = map["icon"] as? FlutterStandardTypedData {
            icon = UIImage(data: data.data)
        }
        var iconColor: UIColor? = nil
        if let hexString = map["iconColor"] as? String {
            iconColor = UIColor(hexString: hexString)
        }
        let showAsAction = map["showAsAction"] as! Bool
        return InAppBrowserMenuItem(id: id, title: title, order: order, icon: icon, iconColor: iconColor, showAsAction: showAsAction)
    }
}
