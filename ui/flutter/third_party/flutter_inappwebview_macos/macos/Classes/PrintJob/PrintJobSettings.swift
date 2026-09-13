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
    public var numberOfPages: Int64? {
        get {
            return _numberOfPages?.int64Value
        }
        set {
            if let newValue = newValue {
                _numberOfPages = NSNumber.init(value: newValue)
            } else {
                _numberOfPages = nil
            }
        }
    }
    public var margins: NSEdgeInsets?
    public var colorMode: String?
    public var showsNumberOfCopies = true
    public var showsPaperOrientation = true
    public var showsPaperSize = true
    public var showsScaling = true
    public var showsPageRange = true
    public var showsPageSetupAccessory = true
    public var showsPreview = true
    public var showsPrintSelection = true
    public var showsPrintPanel = true
    public var showsProgressPanel = true
    public var _scalingFactor: NSNumber?
    public var scalingFactor: Double? {
        get {
            return _scalingFactor?.doubleValue
        }
        set {
            if let newValue = newValue {
                _scalingFactor = NSNumber.init(value: newValue)
            } else {
                _scalingFactor = nil
            }
        }
    }
    public var jobDisposition: String?
    public var jobSavingURL: String?
    public var paperName: String?
    public var _horizontalPagination: NSNumber?
    public var horizontalPagination: UInt? {
        get {
            return _horizontalPagination?.uintValue
        }
        set {
            if let newValue = newValue {
                _horizontalPagination = NSNumber.init(value: newValue)
            } else {
                _horizontalPagination = nil
            }
        }
    }
    public var _verticalPagination: NSNumber?
    public var verticalPagination: UInt? {
        get {
            return _verticalPagination?.uintValue
        }
        set {
            if let newValue = newValue {
                _verticalPagination = NSNumber.init(value: newValue)
            } else {
                _verticalPagination = nil
            }
        }
    }
    public var isHorizontallyCentered = true
    public var isVerticallyCentered = true
    public var _pageOrder: NSNumber?
    public var pageOrder: Int? {
        get {
            return _pageOrder?.intValue
        }
        set {
            if let newValue = newValue {
                _pageOrder = NSNumber.init(value: newValue)
            } else {
                _pageOrder = nil
            }
        }
    }
    public var canSpawnSeparateThread = true
    public var copies = 1
    public var _firstPage: NSNumber?
    public var firstPage: Int64? {
        get {
            return _firstPage?.int64Value
        }
        set {
            if let newValue = newValue {
                _firstPage = NSNumber.init(value: newValue)
            } else {
                _firstPage = nil
            }
        }
    }
    public var _lastPage: NSNumber?
    public var lastPage: Int64? {
        get {
            return _lastPage?.int64Value
        }
        set {
            if let newValue = newValue {
                _lastPage = NSNumber.init(value: newValue)
            } else {
                _lastPage = nil
            }
        }
    }
    public var detailedErrorReporting = false
    public var faxNumber: String?
    public var headerAndFooter = true
    public var _mustCollate: NSNumber?
    public var mustCollate: Bool? {
        get {
            return _mustCollate?.boolValue
        }
        set {
            if let newValue = newValue {
                _mustCollate = NSNumber.init(value: newValue)
            } else {
                _mustCollate = nil
            }
        }
    }
    public var _pagesAcross: NSNumber?
    public var pagesAcross: Int64? {
        get {
            return _pagesAcross?.int64Value
        }
        set {
            if let newValue = newValue {
                _pagesAcross = NSNumber.init(value: newValue)
            } else {
                _pagesAcross = nil
            }
        }
    }
    public var _pagesDown: NSNumber?
    public var pagesDown: Int64? {
        get {
            return _pagesDown?.int64Value
        }
        set {
            if let newValue = newValue {
                _pagesDown = NSNumber.init(value: newValue)
            } else {
                _pagesDown = nil
            }
        }
    }
    public var _time: NSNumber?
    public var time: Int64? {
        get {
            return _time?.int64Value
        }
        set {
            if let newValue = newValue {
                _time = NSNumber.init(value: newValue)
            } else {
                _time = nil
            }
        }
    }
    
    override init(){
        super.init()
    }
    
    override func parse(settings: [String: Any?]) -> PrintJobSettings {
        let _ = super.parse(settings: settings)
        if let marginsMap = settings["margins"] as? [String : Double] {
            margins = NSEdgeInsets.fromMap(map: marginsMap)
        }
        return self
    }
    
    override func getRealSettings(obj: PrintJobController?) -> [String: Any?] {
        var realOptions: [String: Any?] = toMap()
        return realOptions
    }
}
