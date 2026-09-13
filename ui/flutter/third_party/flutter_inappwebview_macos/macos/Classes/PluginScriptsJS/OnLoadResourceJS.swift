//
//  resourceObserverJS.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 16/02/21.
//

import Foundation

let ON_LOAD_RESOURCE_JS_PLUGIN_SCRIPT_GROUP_NAME = "IN_APP_WEBVIEW_ON_LOAD_RESOURCE_JS_PLUGIN_SCRIPT"
let FLAG_VARIABLE_FOR_ON_LOAD_RESOURCE_JS_SOURCE = "window.\(JAVASCRIPT_BRIDGE_NAME)._useOnLoadResource"

let ON_LOAD_RESOURCE_JS_PLUGIN_SCRIPT = PluginScript(
    groupName: ON_LOAD_RESOURCE_JS_PLUGIN_SCRIPT_GROUP_NAME,
    source: ON_LOAD_RESOURCE_JS_SOURCE,
    injectionTime: .atDocumentStart,
    forMainFrameOnly: false,
    requiredInAllContentWorlds: false,
    messageHandlerNames: [])

let ON_LOAD_RESOURCE_JS_SOURCE = """
\(FLAG_VARIABLE_FOR_ON_LOAD_RESOURCE_JS_SOURCE) = true;
(function() {
    var observer = new PerformanceObserver(function(list) {
        list.getEntries().forEach(function(entry) {
            if (\(FLAG_VARIABLE_FOR_ON_LOAD_RESOURCE_JS_SOURCE) == null || \(FLAG_VARIABLE_FOR_ON_LOAD_RESOURCE_JS_SOURCE) == true) {
                var resource = {
                    "url": entry.name,
                    "initiatorType": entry.initiatorType,
                    "startTime": entry.startTime,
                    "duration": entry.duration
                };
                window.\(JAVASCRIPT_BRIDGE_NAME).callHandler("onLoadResource", resource);
            }
        });
    });
    observer.observe({entryTypes: ['resource']});
})();
"""
