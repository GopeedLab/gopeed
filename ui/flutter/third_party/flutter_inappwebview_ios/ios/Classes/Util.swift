//
//  Util.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 12/02/21.
//

import Foundation
import WebKit

var SharedLastTouchPointTimestamp: [InAppWebView: Int64] = [:]

public class Util {
    public static func getUrlAsset(plugin: SwiftFlutterPlugin, assetFilePath: String) throws -> URL {
        guard let key = plugin.registrar?.lookupKey(forAsset: assetFilePath),
              let assetURL = Bundle.main.url(forResource: key, withExtension: nil) else {
            throw NSError(domain: assetFilePath + " asset file cannot be found!", code: 0)
        }
        return assetURL
    }
    
    public static func getAbsPathAsset(plugin: SwiftFlutterPlugin, assetFilePath: String) throws -> String {
        guard let key = plugin.registrar?.lookupKey(forAsset: assetFilePath),
              let assetAbsPath = Bundle.main.path(forResource: key, ofType: nil) else {
            throw NSError(domain: assetFilePath + " asset file cannot be found!", code: 0)
        }
        return assetAbsPath
    }
    
    public static func convertToDictionary(text: String) -> [String: Any]? {
        if let data = text.data(using: .utf8) {
            do {
                return try JSONSerialization.jsonObject(with: data, options: []) as? [String: Any]
            } catch {
                print(error.localizedDescription)
            }
        }
        return nil
    }

    public static func JSONStringify(value: Any, prettyPrinted: Bool = false) -> String {
        let options: JSONSerialization.WritingOptions = prettyPrinted ? .prettyPrinted : .init(rawValue: 0)
        if JSONSerialization.isValidJSONObject(value) {
            let data = try? JSONSerialization.data(withJSONObject: value, options: options)
            if data != nil {
                if let string = String(data: data!, encoding: .utf8) {
                    return string
                }
            }
        }
        return ""
    }
    
    @available(iOS 14.0, *)
    public static func getContentWorld(name: String) -> WKContentWorld {
        switch name {
        case "defaultClient":
            return WKContentWorld.defaultClient
        case "page":
            return WKContentWorld.page
        default:
            return WKContentWorld.world(name: name)
        }
    }
    
    @available(iOS 10.0, *)
    public static func getDataDetectorType(type: String) -> WKDataDetectorTypes {
        switch type {
            case "NONE":
                return WKDataDetectorTypes.init(rawValue: 0)
            case "PHONE_NUMBER":
                return .phoneNumber
            case "LINK":
                return .link
            case "ADDRESS":
                return .address
            case "CALENDAR_EVENT":
                return .calendarEvent
            case "TRACKING_NUMBER":
                return .trackingNumber
            case "FLIGHT_NUMBER":
                return .flightNumber
            case "LOOKUP_SUGGESTION":
                return .lookupSuggestion
            case "SPOTLIGHT_SUGGESTION":
                return .spotlightSuggestion
            case "ALL":
                return .all
            default:
                return WKDataDetectorTypes.init(rawValue: 0)
        }
    }
    
    @available(iOS 10.0, *)
    public static func getDataDetectorTypeString(type: WKDataDetectorTypes) -> [String] {
        var dataDetectorTypeString: [String] = []
        if type.contains(.all) {
            dataDetectorTypeString.append("ALL")
        } else {
            if type.contains(.phoneNumber) {
                dataDetectorTypeString.append("PHONE_NUMBER")
            }
            if type.contains(.link) {
                dataDetectorTypeString.append("LINK")
            }
            if type.contains(.address) {
                dataDetectorTypeString.append("ADDRESS")
            }
            if type.contains(.calendarEvent) {
                dataDetectorTypeString.append("CALENDAR_EVENT")
            }
            if type.contains(.trackingNumber) {
                dataDetectorTypeString.append("TRACKING_NUMBER")
            }
            if type.contains(.flightNumber) {
                dataDetectorTypeString.append("FLIGHT_NUMBER")
            }
            if type.contains(.lookupSuggestion) {
                dataDetectorTypeString.append("LOOKUP_SUGGESTION")
            }
            if type.contains(.spotlightSuggestion) {
                dataDetectorTypeString.append("SPOTLIGHT_SUGGESTION")
            }
        }
        if dataDetectorTypeString.count == 0 {
            dataDetectorTypeString = ["NONE"]
        }
        return dataDetectorTypeString
    }
    
    public static func getDecelerationRate(type: String) -> UIScrollView.DecelerationRate {
        switch type {
            case "NORMAL":
                return .normal
            case "FAST":
                return .fast
            default:
                return .normal
        }
    }
    
    public static func getDecelerationRateString(type: UIScrollView.DecelerationRate) -> String {
        switch type {
            case .normal:
                return "NORMAL"
            case .fast:
                return "FAST"
            default:
                return "NORMAL"
        }
    }
    
    public static func isIPv4(address: String) -> Bool {
        var sin = sockaddr_in()
        return address.withCString({ cstring in inet_pton(AF_INET, cstring, &sin.sin_addr) }) == 1
    }

    public static func isIPv6(address: String) -> Bool {
        var sin6 = sockaddr_in6()
        return address.withCString({ cstring in inet_pton(AF_INET6, cstring, &sin6.sin6_addr) }) == 1
    }

    public static func isIpAddress(address: String) -> Bool {
        return Util.isIPv6(address: address) || Util.isIPv4(address: address)
    }
    
    public static func normalizeIPv6(address: String) throws -> String {
        if !Util.isIPv6(address: address) {
            throw NSError(domain: "Invalid address: \(address)", code: 0)
        }
        var ipString = address
        // replace ipv4 address if any
        let ipv4Regex = try! NSRegularExpression(pattern: "(.*:)([0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+$)")
        if let match = ipv4Regex.firstMatch(in: address, options: [], range: NSRange(location: 0, length: address.utf16.count)) {
            if let ipv6PartRange = Range(match.range(at: 1), in: address) {
                ipString = String(address[ipv6PartRange])
            }
            if let ipv4Range = Range(match.range(at: 2), in: address) {
                let ipv4 = address[ipv4Range]
                let ipv4Split = ipv4.split(separator: ".")
                var ipv4Converted = Array(repeating: "0000", count: 4)
                for i in 0...3 {
                    let byte = Int(ipv4Split[i])!
                    let hex = ("0" + String(byte, radix: 16))
                    var offset = hex.count - 3
                    offset = offset < 0 ? 0 : offset
                    let fromIndex = hex.index(hex.startIndex, offsetBy: offset)
                    let toIndex = hex.index(hex.startIndex, offsetBy: hex.count - 1)
                    let indexRange = Range<String.Index>(uncheckedBounds: (lower: fromIndex, upper: toIndex))
                    ipv4Converted[i] = String(hex[indexRange])
                }
                ipString += ipv4Converted[0] + ipv4Converted[1] + ":" + ipv4Converted[2] + ipv4Converted[3]
            }
        }
        
        // take care of leading and trailing ::
        let regex = try! NSRegularExpression(pattern: "^:|:$")
        ipString = regex.stringByReplacingMatches(in: ipString, options: [], range: NSRange(location: 0, length: ipString.count), withTemplate: "")
        
        let ipv6 = ipString.split(separator: ":", omittingEmptySubsequences: false)
        var fullIPv6 = Array(repeating: "0000", count: ipv6.count)
        
        for (i, hex) in ipv6.enumerated() {
            if !hex.isEmpty {
                // normalize leading zeros
                let hexString = String("0000" + hex)
                var offset = hexString.count - 5
                offset = offset < 0 ? 0 : offset
                let fromIndex = hexString.index(hexString.startIndex, offsetBy: offset)
                let toIndex = hexString.index(hexString.startIndex, offsetBy: hexString.count - 1)
                let indexRange = Range<String.Index>(uncheckedBounds: (lower: fromIndex, upper: toIndex))
                fullIPv6[i] = String(hexString[indexRange])
            } else {
                // normalize grouped zeros ::
                var zeros: [String] = []
                for _ in ipv6.count...8 {
                    zeros.append("0000")
                }
                fullIPv6[i] = zeros.joined(separator: ":")
            }
        }
        
        return fullIPv6.joined(separator: ":")
    }

}
