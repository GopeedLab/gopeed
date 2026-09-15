//
//  PrintAttributes.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 10/05/22.
//

import Foundation

public class PrintAttributes: NSObject {
    var orientation: NSPrintInfo.PaperOrientation?
    var margins: NSEdgeInsets?
    var paperRect: CGRect?
    var colorMode: String?
    var duplex: Int?
    var paperName: String?
    var localizedPaperName: String?
    var horizontalPagination: UInt?
    var verticalPagination: UInt?
    var jobDisposition: String?
    var printableRect: NSRect?
    var isHorizontallyCentered: Bool?
    var isVerticallyCentered: Bool?
    var isSelectionOnly: Bool?
    var scalingFactor: CGFloat?
    var jobSavingURL: String?
    var detailedErrorReporting: Bool?
    var faxNumber: String?
    var headerAndFooter: Bool?
    var mustCollate: Bool?
    var pagesAcross: Int?
    var pagesDown: Int?
    var time: Int?
    
    public init(fromPrintJobController: PrintJobController) {
        super.init()
        if let job = fromPrintJobController.job {
            let printInfo = job.printInfo
            let printInfoDictionary = printInfo.dictionary()
            orientation = printInfo.orientation
            margins = NSEdgeInsets(top: printInfo.topMargin,
                                   left: printInfo.leftMargin,
                                   bottom: printInfo.bottomMargin,
                                   right: printInfo.rightMargin)
            paperRect = CGRect(origin: CGPoint(x: 0.0, y: 0.0), size: printInfo.paperSize)
            colorMode = printInfo.printSettings["ColorModel"] as? String
            duplex = printInfo.printSettings["com_apple_print_PrintSettings_PMDuplexing"] as? Int
            paperName = printInfo.paperName?.rawValue
            localizedPaperName = printInfo.localizedPaperName
            horizontalPagination = printInfo.horizontalPagination.rawValue
            verticalPagination = printInfo.verticalPagination.rawValue
            jobDisposition = printInfo.jobDisposition.rawValue
            printableRect = printInfo.imageablePageBounds
            isHorizontallyCentered = printInfo.isHorizontallyCentered
            isVerticallyCentered = printInfo.isVerticallyCentered
            isSelectionOnly = printInfo.isSelectionOnly
            scalingFactor = printInfo.scalingFactor
            jobSavingURL = (printInfoDictionary[NSPrintInfo.AttributeKey.jobSavingURL] as? URL)?.absoluteString
            detailedErrorReporting = printInfoDictionary[NSPrintInfo.AttributeKey.detailedErrorReporting] as? Bool
            faxNumber = printInfoDictionary[NSPrintInfo.AttributeKey.faxNumber] as? String
            headerAndFooter = printInfoDictionary[NSPrintInfo.AttributeKey.headerAndFooter] as? Bool
            mustCollate = printInfoDictionary[NSPrintInfo.AttributeKey.mustCollate] as? Bool
            pagesAcross = printInfoDictionary[NSPrintInfo.AttributeKey.pagesAcross] as? Int
            pagesDown = printInfoDictionary[NSPrintInfo.AttributeKey.pagesDown] as? Int
            if let timestamp = (printInfoDictionary[NSPrintInfo.AttributeKey.time] as? Date)?.timeIntervalSince1970 {
                time = Int(timestamp)
            }
        }
    }
    
    public func toMap () -> [String:Any?] {
        return [
            "paperRect": paperRect?.toMap(),
            "margins": margins?.toMap(),
            "orientation": orientation?.rawValue,
            "colorMode": colorMode,
            "duplex": duplex,
            "paperName": paperName,
            "localizedPaperName": localizedPaperName,
            "horizontalPagination": horizontalPagination,
            "verticalPagination": verticalPagination,
            "jobDisposition": jobDisposition,
            "printableRect": printableRect?.toMap(),
            "isHorizontallyCentered": isHorizontallyCentered,
            "isVerticallyCentered": isVerticallyCentered,
            "isSelectionOnly": isSelectionOnly,
            "scalingFactor": scalingFactor,
            "jobSavingURL": jobSavingURL,
            "detailedErrorReporting": detailedErrorReporting,
            "faxNumber": faxNumber,
            "headerAndFooter": headerAndFooter,
            "mustCollate": mustCollate,
            "pagesAcross": pagesAcross,
            "pagesDown": pagesDown,
            "time": time
        ]
    }
}
