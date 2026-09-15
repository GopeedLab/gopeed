//
//  NSAttributedString.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 05/03/21.
//

import Foundation

extension NSAttributedString {
    public static func fromMap(map: [String:Any?]?) -> NSAttributedString? {
        guard let map = map, let string = map["string"] as? String else {
            return nil
        }
        
        var attributes: [NSAttributedString.Key : Any] = [:]
        
        if let backgroundColor = map["backgroundColor"] as? String {
            attributes[.backgroundColor] = UIColor(hexString: backgroundColor)
        }
        if let baselineOffset = map["baselineOffset"] as? Double {
            attributes[.baselineOffset] = baselineOffset
        }
        if let expansion = map["expansion"] as? Double {
            attributes[.expansion] = expansion
        }
        if let foregroundColor = map["foregroundColor"] as? String {
            attributes[.foregroundColor] = UIColor(hexString: foregroundColor)
        }
        if let kern = map["kern"] as? Double {
            attributes[.kern] = kern
        }
        if let ligature = map["ligature"] as? Int64 {
            attributes[.ligature] = ligature
        }
        if let obliqueness = map["obliqueness"] as? Double {
            attributes[.obliqueness] = obliqueness
        }
        if let strikethroughColor = map["strikethroughColor"] as? String {
            attributes[.strikethroughColor] = UIColor(hexString: strikethroughColor)
        }
        if let strikethroughStyle = map["strikethroughStyle"] as? Int64 {
            attributes[.strikethroughStyle] = strikethroughStyle
        }
        if let strokeColor = map["strokeColor"] as? String {
            attributes[.strokeColor] = UIColor(hexString: strokeColor)
        }
        if let strokeWidth = map["strokeWidth"] as? Double {
            attributes[.strokeWidth] = strokeWidth
        }
        if let textEffect = map["textEffect"] as? String {
            attributes[.textEffect] = textEffect
        }
        if let underlineColor = map["underlineColor"] as? String {
            attributes[.underlineColor] = UIColor(hexString: underlineColor)
        }
        if let underlineStyle = map["underlineStyle"] as? Int64 {
            attributes[.underlineStyle] = underlineStyle
        }
        
        return NSAttributedString(string: string, attributes: attributes)
    }
}
