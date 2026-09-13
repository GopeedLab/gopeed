//
//  PrintJobInfo.swift
//  flutter_downloader
//
//  Created by Lorenzo Pichilli on 10/05/22.
//

import Foundation

public class PrintJobInfo: NSObject {
    var state: PrintJobState
    var attributes: PrintAttributes
    var creationTime: Int64
    var numberOfPages: Int?
    var copies: Int?
    var label: String?
    var printer: NSPrinter?
    var pageOrder: Int?
    var preferredRenderingQuality: Int?
    var showsProgressPanel: Bool?
    var showsPrintPanel: Bool?
    var canSpawnSeparateThread: Bool?
    var isCopyingOperation: Bool?
    var currentPage: Int?
    var firstPage: Int?
    var lastPage: Int?
    
    public init(fromPrintJobController: PrintJobController) {
        state = fromPrintJobController.state
        creationTime = fromPrintJobController.creationTime
        attributes = PrintAttributes.init(fromPrintJobController: fromPrintJobController)
        super.init()
        if let job = fromPrintJobController.job {
            let printInfo = job.printInfo
            let printInfoDictionary = printInfo.dictionary()
            printer = printInfo.printer
            copies = printInfo.printSettings["com_apple_print_PrintSettings_PMCopies"] as? Int
            label = job.jobTitle
            firstPage = printInfoDictionary[NSPrintInfo.AttributeKey.firstPage] as? Int
            lastPage = printInfoDictionary[NSPrintInfo.AttributeKey.lastPage] as? Int
            if let firstPage = firstPage, let lastPage = lastPage {
                numberOfPages = lastPage - firstPage + 1
            }
            if numberOfPages == nil {
                numberOfPages = printInfo.printSettings["com_apple_print_PrintSettings_PMLastPage"] as? Int
                if numberOfPages == nil || numberOfPages! > job.pageRange.length {
                    numberOfPages = job.pageRange.length
                }
            }
            pageOrder = job.pageOrder.rawValue
            preferredRenderingQuality = job.preferredRenderingQuality.rawValue
            showsProgressPanel = job.showsProgressPanel
            showsPrintPanel = job.showsPrintPanel
            canSpawnSeparateThread = job.canSpawnSeparateThread
            isCopyingOperation = job.isCopyingOperation
            currentPage = job.currentPage
        }
    }
    
    public func toMap () -> [String:Any?] {
        return [
            "state": state.rawValue,
            "attributes": attributes.toMap(),
            "numberOfPages": numberOfPages,
            "copies": copies,
            "creationTime": creationTime,
            "label": label,
            "printer": printer?.toMap(),
            "pageOrder": pageOrder,
            "preferredRenderingQuality": preferredRenderingQuality,
            "showsProgressPanel": showsProgressPanel,
            "showsPrintPanel": showsPrintPanel,
            "canSpawnSeparateThread": canSpawnSeparateThread,
            "isCopyingOperation": isCopyingOperation,
            "currentPage": currentPage,
            "firstPage": firstPage,
            "lastPage": lastPage
        ]
    }
}
