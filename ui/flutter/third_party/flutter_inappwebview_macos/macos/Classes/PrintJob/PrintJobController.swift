//
//  PrintJob.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 09/05/22.
//

import Foundation
import FlutterMacOS

public enum PrintJobState: Int {
    case created = 1
    case started = 3
    case completed = 5
    case canceled = 7
}

public class PrintJobController: NSObject, Disposable {
    static let METHOD_CHANNEL_NAME_PREFIX = "com.pichillilorenzo/flutter_inappwebview_printjobcontroller_"
    var id: String
    var plugin: InAppWebViewFlutterPlugin?
    var job: NSPrintOperation?
    var settings: PrintJobSettings?
    var channelDelegate: PrintJobChannelDelegate?
    var state = PrintJobState.created
    var creationTime = Int64(Date().timeIntervalSince1970 * 1000)
    private var completionHandler: PrintJobController.CompletionHandler?
    
    public typealias CompletionHandler = (_ printOperation: NSPrintOperation,
                                          _ success: Bool,
                                          _ contextInfo: UnsafeMutableRawPointer?) -> Void
    
    public init(plugin: InAppWebViewFlutterPlugin, id: String, job: NSPrintOperation? = nil, settings: PrintJobSettings? = nil) {
        self.id = id
        self.plugin = plugin
        super.init()
        self.job = job
        self.settings = settings
        if let registrar = plugin.registrar {
            let channel = FlutterMethodChannel(name: PrintJobController.METHOD_CHANNEL_NAME_PREFIX + id,
                                               binaryMessenger: registrar.messenger)
            self.channelDelegate = PrintJobChannelDelegate(printJobController: self, channel: channel)
        }
    }
    
    public func present(parentWindow: NSWindow? = nil, completionHandler: PrintJobController.CompletionHandler? = nil) {
        guard let job = job else {
            return
        }
        state = .started
        self.completionHandler = completionHandler
        if let mainWindow = parentWindow ?? NSApplication.shared.mainWindow {
            job.runModal(for: mainWindow, delegate: self, didRun: #selector(printOperationDidRun), contextInfo: nil)
        }
    }
    
    @objc func printOperationDidRun(printOperation: NSPrintOperation,
                                    success: Bool,
                                    contextInfo: UnsafeMutableRawPointer?) {
        state = success ? .completed : .canceled
        channelDelegate?.onComplete(completed: success, error: nil)
        if let completionHandler = completionHandler {
            completionHandler(printOperation, success, contextInfo)
            self.completionHandler = nil
        }
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
        completionHandler = nil
        job = nil
        plugin?.printJobManager?.jobs[id] = nil
        plugin = nil
    }
    
    public func dispose() {
        channelDelegate?.dispose()
        channelDelegate = nil
        completionHandler = nil
        job = nil
        plugin?.printJobManager?.jobs[id] = nil
        plugin = nil
    }
}
