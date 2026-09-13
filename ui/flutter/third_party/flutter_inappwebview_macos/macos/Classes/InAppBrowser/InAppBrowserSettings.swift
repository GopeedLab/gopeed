//
//  InAppBrowserOptions.swift
//  flutter_inappwebview
//
//  Created by Lorenzo on 17/09/18.
//

import Foundation

@objcMembers
public class InAppBrowserSettings: ISettings<InAppBrowserWebViewController> {
    
    var hidden = false
    var hideToolbarTop = true
    var toolbarTopBackgroundColor: String?
    var hideUrlBar = false
    var hideProgressBar = false
    var toolbarTopFixedTitle: String?
    var windowType = InAppBrowserWindowType.window
    var windowAlphaValue = 1.0
    var _windowStyleMask: NSNumber?
    var windowStyleMask: NSWindow.StyleMask? {
        get {
            return _windowStyleMask != nil ?
                    NSWindow.StyleMask.init(rawValue: _windowStyleMask!.uintValue) :
                    nil
        }
        set {
            if let newValue = newValue {
                _windowStyleMask = NSNumber.init(value: newValue.rawValue)
            } else {
                _windowStyleMask = nil
            }
        }
    }
    var _windowTitlebarSeparatorStyle: NSNumber?
    @available(macOS 11.0, *)
    var windowTitlebarSeparatorStyle: NSTitlebarSeparatorStyle? {
        get {
            return _windowTitlebarSeparatorStyle != nil ?
                    NSTitlebarSeparatorStyle.init(rawValue: _windowTitlebarSeparatorStyle!.intValue) :
                    nil
        }
        set {
            if let newValue = newValue {
                _windowTitlebarSeparatorStyle = NSNumber.init(value: newValue.rawValue)
            } else {
                _windowTitlebarSeparatorStyle = nil
            }
        }
    }
    var windowFrame: NSRect?
    var hideDefaultMenuItems = false
    
    override init(){
        super.init()
    }
    
    override func parse(settings: [String: Any?]) -> InAppBrowserSettings {
        let _ = super.parse(settings: settings)
        if let windowType = settings["windowType"] as? String {
            self.windowType = InAppBrowserWindowType.init(rawValue: windowType) ?? InAppBrowserWindowType.child
        }
        if let windowFrame = settings["windowFrame"] as? [String:Double] {
            self.windowFrame = NSRect(x: windowFrame["x"]!,
                                      y: windowFrame["y"]!,
                                      width: windowFrame["width"]!,
                                      height: windowFrame["height"]!)
        }
        return self
    }
    
    override func getRealSettings(obj: InAppBrowserWebViewController?) -> [String: Any?] {
        var realOptions: [String: Any?] = toMap()
        realOptions["windowType"] = windowType.rawValue
        if let inAppBrowserWebViewController = obj {
            realOptions["hidden"] = inAppBrowserWebViewController.isHidden
            realOptions["hideUrlBar"] = inAppBrowserWebViewController.window?.searchBar?.isHidden
            realOptions["hideProgressBar"] = inAppBrowserWebViewController.progressBar.isHidden
            realOptions["hideToolbarTop"] = !(inAppBrowserWebViewController.window?.toolbar?.isVisible ?? true)
            realOptions["toolbarTopBackgroundColor"] = inAppBrowserWebViewController.window?.backgroundColor.hexString
            realOptions["windowAlphaValue"] = inAppBrowserWebViewController.window?.alphaValue
            realOptions["windowStyleMask"] = inAppBrowserWebViewController.window?.styleMask.rawValue
            if #available(macOS 11.0, *) {
                realOptions["windowTitlebarSeparatorStyle"] = inAppBrowserWebViewController.window?.titlebarSeparatorStyle.rawValue
            }
            realOptions["windowFrame"] = inAppBrowserWebViewController.window?.frame.toMap()
        }
        return realOptions
    }
}
