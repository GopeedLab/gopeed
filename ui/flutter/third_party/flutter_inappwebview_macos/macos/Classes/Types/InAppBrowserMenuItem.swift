//
//  InAppBrowserMenuItem.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 23/05/23.
//

import Foundation
import FlutterMacOS

public class InAppBrowserMenuItem: NSObject {
    var id: Int64
    var title: String
    var order: Int64?
    var icon: NSImage?
    var iconColor: NSColor?
    var showAsAction = false
    
    public init(id: Int64, title: String, order: Int64?, icon: NSImage?, iconColor: NSColor?, showAsAction: Bool) {
        self.id = id
        self.title = title
        self.order = order
        self.icon = icon
        self.iconColor = iconColor
        self.showAsAction = showAsAction
        if let icon = icon, let iconColor = iconColor {
            self.icon = icon.tint(color: iconColor)
        }
    }
    
    public static func fromMap(map: [String:Any?]?) -> InAppBrowserMenuItem? {
        guard let map = map else {
            return nil
        }
        let id = map["id"] as! Int64
        let title = map["title"] as! String
        let order = map["order"] as? Int64
        var icon = NSImage.fromMap(map: map["icon"] as? [String : Any?])
        if let data = map["icon"] as? FlutterStandardTypedData {
            icon = NSImage(data: data.data)
        }
        var iconColor: NSColor? = nil
        if let hexString = map["iconColor"] as? String {
            iconColor = NSColor(hexString: hexString)
        }
        let showAsAction = map["showAsAction"] as! Bool
        return InAppBrowserMenuItem(id: id, title: title, order: order, icon: icon, iconColor: iconColor, showAsAction: showAsAction)
    }
}
