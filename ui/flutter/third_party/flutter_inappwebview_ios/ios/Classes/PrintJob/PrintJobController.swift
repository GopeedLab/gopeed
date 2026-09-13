//
//  PrintJob.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 09/05/22.
//

import Foundation

public enum PrintJobState: Int {
    case created = 1
    case started = 3
    case completed = 5
    case failed = 6
    case canceled = 7
}

public class PrintJobController: NSObject, Disposable, UIPrintInteractionControllerDelegate {
    static let METHOD_CHANNEL_NAME_PREFIX = "com.pichillilorenzo/flutter_inappwebview_printjobcontroller_"
    var id: String
    var plugin: SwiftFlutterPlugin?
    var job: UIPrintInteractionController?
    var settings: PrintJobSettings?
    var printFormatter: UIPrintFormatter?
    var printPageRenderer: UIPrintPageRenderer?
    var channelDelegate: PrintJobChannelDelegate?
    var state = PrintJobState.created
    var creationTime = Int64(Date().timeIntervalSince1970 * 1000)
    
    public init(plugin: SwiftFlutterPlugin, id: String, job: UIPrintInteractionController? = nil, settings: PrintJobSettings? = nil) {
        self.id = id
        self.plugin = plugin
        super.init()
        self.job = job
        self.settings = settings
        self.printFormatter = job?.printFormatter
        self.printPageRenderer = job?.printPageRenderer
        self.job?.delegate = self
        if let registrar = plugin.registrar {
            let channel = FlutterMethodChannel(name: PrintJobController.METHOD_CHANNEL_NAME_PREFIX + id,
                                               binaryMessenger: registrar.messenger())
            self.channelDelegate = PrintJobChannelDelegate(printJobController: self, channel: channel)
        }
    }
    
    public func printInteractionControllerWillStartJob(_ printInteractionController: UIPrintInteractionController) {
        state = .started
    }
    
    public func present(animated: Bool, completionHandler: UIPrintInteractionController.CompletionHandler? = nil) {
        guard let job = job else {
            return
        }
        
        job.present(animated: animated, completionHandler: { [weak self] (printController, completed, error) in
            if !completed {
                if let _ = error {
                    self?.state = .failed
                } else {
                    self?.state = .canceled
                }
            } else {
                self?.state = .completed
            }
            self?.channelDelegate?.onComplete(completed: completed, error: error)
            if let completionHandler = completionHandler {
                completionHandler(printController, completed, error)
            }
        })
    }
    
    public func getInfo() -> PrintJobInfo? {
        guard let _ = job else {
            return nil
        }
        
        return PrintJobInfo.init(fromPrintJobController: self)
    }
    
    public func disposeNoDismiss() {
        channelDelegate?.dispose()
        channelDelegate = nil
        printFormatter = nil
        printPageRenderer = nil
        job?.delegate = nil
        job = nil
        plugin?.printJobManager?.jobs[id] = nil
        plugin = nil
    }
    
    public func dispose() {
        channelDelegate?.dispose()
        channelDelegate = nil
        printFormatter = nil
        printPageRenderer = nil
        job?.delegate = nil
        job?.dismiss(animated: false)
        job = nil
        plugin?.printJobManager?.jobs[id] = nil
        plugin = nil
    }
}
