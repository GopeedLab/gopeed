//
//  PlatformUtil.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 01/03/21.
//

import Foundation

public class PlatformUtil: ChannelDelegate {
    static let METHOD_CHANNEL_NAME = "com.pichillilorenzo/flutter_inappwebview_platformutil"
    var plugin: SwiftFlutterPlugin?
    
    init(plugin: SwiftFlutterPlugin) {
        super.init(channel: FlutterMethodChannel(name: PlatformUtil.METHOD_CHANNEL_NAME, binaryMessenger: plugin.registrar!.messenger()))
        self.plugin = plugin
    }
    
    public override func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
        let arguments = call.arguments as? NSDictionary
        
        switch call.method {
            case "getSystemVersion":
                let device = UIDevice.current
                result(device.systemVersion)
                break
            case "formatDate":
                let date = arguments!["date"] as! Int64
                let format = arguments!["format"] as! String
                let locale = PlatformUtil.getLocaleFromString(locale: arguments!["locale"] as? String)
                let timezone = TimeZone.init(abbreviation: arguments!["timezone"] as? String ?? "UTC")!
                result(PlatformUtil.formatDate(date: date, format: format, locale: locale, timezone: timezone))
                break
            default:
                result(FlutterMethodNotImplemented)
                break
        }
    }
    
    static public func getLocaleFromString(locale: String?) -> Locale {
        guard let locale = locale else {
            return Locale.init(identifier: "en_US")
        }
        return Locale.init(identifier: locale)
    }
    
    static public func getDateFromMilliseconds(date: Int64) -> Date {
        return Date(timeIntervalSince1970: TimeInterval(Double(date)/1000))
    }
    
    static public func formatDate(date: Int64, format: String, locale: Locale, timezone: TimeZone) -> String {
        let formatter = DateFormatter()
        formatter.dateFormat = format
        formatter.timeZone = timezone
        return formatter.string(from: PlatformUtil.getDateFromMilliseconds(date: date))
    }
    
    public override func dispose() {
        super.dispose()
        plugin = nil
    }
    
    deinit {
        dispose()
    }
}
