//
//  PrintJS.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 16/02/21.
//

import Foundation

let PRINT_JS_PLUGIN_SCRIPT_GROUP_NAME = "IN_APP_WEBVIEW_PRINT_JS_PLUGIN_SCRIPT"

let PRINT_JS_PLUGIN_SCRIPT = PluginScript(
    groupName: PRINT_JS_PLUGIN_SCRIPT_GROUP_NAME,
    source: PRINT_JS_SOURCE,
    injectionTime: .atDocumentStart,
    forMainFrameOnly: false,
    requiredInAllContentWorlds: true,
    messageHandlerNames: [])

let PRINT_JS_SOURCE = """
window.print = function() {
    window.\(JAVASCRIPT_BRIDGE_NAME).callHandler("onPrintRequest", window.location.href);
}
"""
