//
//  WebMessage.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 10/03/21.
//

import Foundation
import FlutterMacOS

public class WebMessage: NSObject, Disposable {
    var data: Any?
    var type: WebMessageType
    var ports: [WebMessagePort]?
    
    var jsData: String {
        var jsData: String = "null"
        if let messageData = data {
            if type == .arrayBuffer, let messageDataArrayBuffer = messageData as? FlutterStandardTypedData {
                jsData = "new Uint8Array(\(Array(messageDataArrayBuffer.data))).buffer"
            } else if let messageDataString = messageData as? String {
                jsData = "'\(messageDataString.replacingOccurrences(of: "\'", with: "\\'"))'"
            }
        }
        return jsData
    }
    
    public init(data: Any?, type: WebMessageType, ports: [WebMessagePort]?) {
        self.type = type
        super.init()
        self.data = data
        self.ports = ports
    }
    
    public static func fromMap(map: [String: Any?]) -> WebMessage {
        let portMapList = map["ports"] as? [[String: Any?]]
        var ports: [WebMessagePort]? = nil
        if let portMapList = portMapList, !portMapList.isEmpty {
            ports = []
            portMapList.forEach { (portMap) in
                ports?.append(WebMessagePort.fromMap(map: portMap))
            }
        }
        
        return WebMessage(
            data: map["data"] as? Any,
            type: WebMessageType.init(rawValue: map["type"] as! Int)!,
            ports: ports)
    }
    
    public func toMap () -> [String: Any?] {
        return [
            "data": type == .arrayBuffer && data is [UInt8] ? Data(data as! [UInt8]) : data,
            "type": type.rawValue
        ]
    }
    
    public func dispose() {
        ports?.removeAll()
    }
    
    deinit {
        debugPrint("WebMessage - dealloc")
        dispose()
    }
}

public enum WebMessageType: Int {
    case string = 0
    case arrayBuffer = 1
}
