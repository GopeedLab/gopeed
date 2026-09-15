//
//  MyWebStorageManager.swift
//  connectivity
//
//  Created by Lorenzo Pichilli on 16/12/2019.
//

import Foundation
import WebKit

@available(iOS 9.0, *)
public class MyWebStorageManager: ChannelDelegate {
    static let METHOD_CHANNEL_NAME = "com.pichillilorenzo/flutter_inappwebview_webstoragemanager"
    static let websiteDataStore = WKWebsiteDataStore.default()

    private var plugin: SwiftFlutterPlugin?
    
    init(plugin: SwiftFlutterPlugin) {
        super.init(channel: FlutterMethodChannel(name: MyWebStorageManager.METHOD_CHANNEL_NAME, binaryMessenger: plugin.registrar!.messenger()))
        self.plugin = plugin
    }
    
    public override func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
        let arguments = call.arguments as? NSDictionary
        switch call.method {
            case "fetchDataRecords":
                let dataTypes = Set(arguments!["dataTypes"] as! [String])
                MyWebStorageManager.fetchDataRecords(dataTypes: dataTypes, result: result)
                break
            case "removeDataFor":
                let dataTypes = Set(arguments!["dataTypes"] as! [String])
                let recordList = arguments!["recordList"] as! [[String: Any?]]
                MyWebStorageManager.removeDataFor(dataTypes: dataTypes, recordList: recordList, result: result)
                break
            case "removeDataModifiedSince":
                let dataTypes = Set(arguments!["dataTypes"] as! [String])
                let timestamp = arguments!["timestamp"] as! Int64
                MyWebStorageManager.removeDataModifiedSince(dataTypes: dataTypes, timestamp: timestamp, result: result)
                break
            default:
                result(FlutterMethodNotImplemented)
                break
        }
    }
    
    public static func fetchDataRecords(dataTypes: Set<String>, result: @escaping FlutterResult) {
        var recordList: [[String: Any?]] = []

        MyWebStorageManager.websiteDataStore.fetchDataRecords(ofTypes: dataTypes) { (data) in
            for record in data {
                recordList.append([
                    "displayName": record.displayName,
                    "dataTypes": record.dataTypes.map({ (dataType) -> String in
                        return dataType
                    })
                ])
            }
            result(recordList)
        }
    }
    
    public static func removeDataFor(dataTypes: Set<String>, recordList: [[String: Any?]], result: @escaping FlutterResult) {
        var records: [WKWebsiteDataRecord] = []

        MyWebStorageManager.websiteDataStore.fetchDataRecords(ofTypes: dataTypes) { (data) in
            for record in data {
                for r in recordList {
                    let displayName = r["displayName"] as! String
                    if (record.displayName == displayName) {
                        records.append(record)
                        break
                    }
                }
            }
            MyWebStorageManager.websiteDataStore.removeData(ofTypes: dataTypes, for: records) {
                result(true)
            }
        }
    }
    
    public static func removeDataModifiedSince(dataTypes: Set<String>, timestamp: Int64, result: @escaping FlutterResult) {
        let date = NSDate(timeIntervalSince1970: TimeInterval(timestamp))
        MyWebStorageManager.websiteDataStore.removeData(ofTypes: dataTypes, modifiedSince: date as Date) {
            result(true)
        }
    }
    
    public override func dispose() {
        super.dispose()
        plugin = nil
    }
    
    deinit {
        dispose()
    }
}
