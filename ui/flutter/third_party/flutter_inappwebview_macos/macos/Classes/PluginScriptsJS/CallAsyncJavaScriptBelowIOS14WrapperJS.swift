//
//  CallAsyncJavaScriptBelowIOS14WrapperJS.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 16/02/21.
//

import Foundation

let CALL_ASYNC_JAVASCRIPT_BELOW_IOS_14_WRAPPER_JS = """
(function(obj) {
    (async function(\(PluginScriptsUtil.VAR_FUNCTION_ARGUMENT_NAMES) {
        \(PluginScriptsUtil.VAR_FUNCTION_BODY)
    })(\(PluginScriptsUtil.VAR_FUNCTION_ARGUMENT_VALUES)).then(function(value) {
        window.webkit.messageHandlers['onCallAsyncJavaScriptResultBelowIOS14Received'].postMessage({'value': value, 'error': null, 'resultUuid': '\(PluginScriptsUtil.VAR_RESULT_UUID)'});
    }).catch(function(error) {
        window.webkit.messageHandlers['onCallAsyncJavaScriptResultBelowIOS14Received'].postMessage({'value': null, 'error': error + '', 'resultUuid': '\(PluginScriptsUtil.VAR_RESULT_UUID)'});
    });
    return null;
})(\(PluginScriptsUtil.VAR_FUNCTION_ARGUMENTS_OBJ));
"""
