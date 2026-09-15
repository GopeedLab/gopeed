//
//  PrintJobSettings.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 09/05/22.
//

import Foundation

@objcMembers
public class PrintJobSettings: ISettings<PrintJobController> {
    
    public var handledByClient = false
    public var jobName: String?
    public var animated = true
    public var _orientation: NSNumber?
    public var orientation: Int? {
        get {
            return _orientation?.intValue
        }
        set {
            if let newValue = newValue {
                _orientation = NSNumber.init(value: newValue)
            } else {
                _orientation = nil
            }
        }
    }
    public var _numberOfPages: NSNumber?
    public var numberOfPages: Int? {
        get {
            return _numberOfPages?.intValue
        }
        set {
            if let newValue = newValue {
                _numberOfPages = NSNumber.init(value: newValue)
            } else {
                _numberOfPages = nil
            }
        }
    }
    public var _forceRenderingQuality: NSNumber?
    public var forceRenderingQuality: Int? {
        get {
            return _forceRenderingQuality?.intValue
        }
        set {
            if let newValue = newValue {
                _forceRenderingQuality = NSNumber.init(value: newValue)
            } else {
                _forceRenderingQuality = nil
            }
        }
    }
    public var margins: UIEdgeInsets?
    public var _duplexMode: NSNumber?
    public var duplexMode: Int? {
        get {
            return _duplexMode?.intValue
        }
        set {
            if let newValue = newValue {
                _duplexMode = NSNumber.init(value: newValue)
            } else {
                _duplexMode = nil
            }
        }
    }
    public var _outputType: NSNumber?
    public var outputType: Int? {
        get {
            return _outputType?.intValue
        }
        set {
            if let newValue = newValue {
                _outputType = NSNumber.init(value: newValue)
            } else {
                _outputType = nil
            }
        }
    }
    public var showsNumberOfCopies = true
    public var showsPaperSelectionForLoadedPapers = false
    public var showsPaperOrientation = true
    public var _maximumContentHeight: NSNumber?
    public var maximumContentHeight: Double? {
        get {
            return _maximumContentHeight?.doubleValue
        }
        set {
            if let newValue = newValue {
                _maximumContentHeight = NSNumber.init(value: newValue)
            } else {
                _maximumContentHeight = nil
            }
        }
    }
    public var _maximumContentWidth: NSNumber?
    public var maximumContentWidth: Double? {
        get {
            return _maximumContentWidth?.doubleValue
        }
        set {
            if let newValue = newValue {
                _maximumContentWidth = NSNumber.init(value: newValue)
            } else {
                _maximumContentWidth = nil
            }
        }
    }
    public var _footerHeight: NSNumber?
    public var footerHeight: Double? {
        get {
            return _footerHeight?.doubleValue
        }
        set {
            if let newValue = newValue {
                _footerHeight = NSNumber.init(value: newValue)
            } else {
                _footerHeight = nil
            }
        }
    }
    public var _headerHeight: NSNumber?
    public var headerHeight: Double? {
        get {
            return _headerHeight?.doubleValue
        }
        set {
            if let newValue = newValue {
                _headerHeight = NSNumber.init(value: newValue)
            } else {
                _headerHeight = nil
            }
        }
    }
    
    override init(){
        super.init()
    }
    
    override func parse(settings: [String: Any?]) -> PrintJobSettings {
        let _ = super.parse(settings: settings)
        if let marginsMap = settings["margins"] as? [String : Double] {
            margins = UIEdgeInsets.fromMap(map: marginsMap)
        }
        return self
    }
    
    override func getRealSettings(obj: PrintJobController?) -> [String: Any?] {
        let realOptions: [String: Any?] = toMap()
        return realOptions
    }
}
