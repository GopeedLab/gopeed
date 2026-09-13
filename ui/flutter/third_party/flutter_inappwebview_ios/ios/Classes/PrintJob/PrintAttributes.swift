//
//  PrintAttributes.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 10/05/22.
//

import Foundation

public class PrintAttributes: NSObject {
    var orientation: UIPrintInfo.Orientation?
    var duplex: UIPrintInfo.Duplex?
    var outputType: UIPrintInfo.OutputType?
    var margins: UIEdgeInsets?
    var footerHeight: Double?
    var headerHeight: Double?
    var paperRect: CGRect?
    var printableRect: CGRect?
    var maximumContentHeight: Double?
    var maximumContentWidth: Double?
    
    public init(fromPrintJobController: PrintJobController) {
        super.init()
        if let printPageRenderer = fromPrintJobController.printPageRenderer {
            footerHeight = printPageRenderer.footerHeight
            headerHeight = printPageRenderer.headerHeight
            paperRect = printPageRenderer.paperRect
            printableRect = printPageRenderer.printableRect
        }
        if let printFormatter = fromPrintJobController.printFormatter {
            maximumContentHeight = printFormatter.maximumContentHeight
            maximumContentWidth = printFormatter.maximumContentWidth
            margins = printFormatter.perPageContentInsets
        }
        if let job = fromPrintJobController.job, let printInfo = job.printInfo {
            orientation = printInfo.orientation
            duplex = printInfo.duplex
            outputType = printInfo.outputType
        }
    }
    
    public func toMap () -> [String:Any?] {
        return [
            "footerHeight": footerHeight,
            "headerHeight": headerHeight,
            "paperRect": paperRect?.toMap(),
            "printableRect": printableRect?.toMap(),
            "margins": margins?.toMap(),
            "maximumContentHeight": maximumContentHeight,
            "maximumContentWidth": maximumContentWidth,
            "orientation": orientation?.rawValue,
            "duplex": duplex?.rawValue,
            "outputType": outputType?.rawValue
        ]
    }
}
