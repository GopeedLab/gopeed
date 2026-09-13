//
//  ConsoleLogJS.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 16/02/21.
//

import Foundation

let CONSOLE_LOG_JS_PLUGIN_SCRIPT_GROUP_NAME = "IN_APP_WEBVIEW_CONSOLE_LOG_JS_PLUGIN_SCRIPT"

let CONSOLE_LOG_JS_PLUGIN_SCRIPT = PluginScript(
    groupName: CONSOLE_LOG_JS_PLUGIN_SCRIPT_GROUP_NAME,
    source: CONSOLE_LOG_JS_SOURCE,
    injectionTime: .atDocumentStart,
    forMainFrameOnly: false,
    requiredInAllContentWorlds: true,
    messageHandlerNames: ["consoleLog", "consoleDebug", "consoleError", "consoleInfo", "consoleWarn"])

// the message needs to be concatenated with '' in order to have the same behavior like on Android
let CONSOLE_LOG_JS_SOURCE = """
(function(console) {

    function _callHandler(oldLog, args) {
        var message = '';
        for (var i in args) {
            try {
                message += message === '' ? args[i] : ' ' + args[i];
            } catch(ignored) {}
        }
        var _windowId = \(WINDOW_ID_VARIABLE_JS_SOURCE);
        window.webkit.messageHandlers[oldLog].postMessage({'message': message, '_windowId': _windowId});
    }

    var oldLogs = {
        'consoleLog': console.log,
        'consoleDebug': console.debug,
        'consoleError': console.error,
        'consoleInfo': console.info,
        'consoleWarn': console.warn
    };

    for (var k in oldLogs) {
        (function(oldLog) {
            console[oldLog.replace('console', '').toLowerCase()] = function() {
                oldLogs[oldLog].apply(null, arguments);
                _callHandler(oldLog, arguments);
            }
        })(k);
    }
})(window.console);
"""
