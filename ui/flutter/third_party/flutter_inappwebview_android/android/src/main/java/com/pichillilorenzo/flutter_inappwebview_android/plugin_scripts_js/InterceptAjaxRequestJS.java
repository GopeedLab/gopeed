package com.pichillilorenzo.flutter_inappwebview_android.plugin_scripts_js;

import com.pichillilorenzo.flutter_inappwebview_android.types.PluginScript;
import com.pichillilorenzo.flutter_inappwebview_android.types.UserScriptInjectionTime;

public class InterceptAjaxRequestJS {

  public static final String INTERCEPT_AJAX_REQUEST_JS_PLUGIN_SCRIPT_GROUP_NAME = "IN_APP_WEBVIEW_INTERCEPT_AJAX_REQUEST_JS_PLUGIN_SCRIPT";
  public static final String FLAG_VARIABLE_FOR_SHOULD_INTERCEPT_AJAX_REQUEST_JS_SOURCE = JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + "._useShouldInterceptAjaxRequest";
  public static final String FLAG_VARIABLE_FOR_INTERCEPT_ONLY_ASYNC_AJAX_REQUESTS_JS_SOURCE = JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + "._interceptOnlyAsyncAjaxRequests";
  public static final PluginScript INTERCEPT_AJAX_REQUEST_JS_PLUGIN_SCRIPT = new PluginScript(
          InterceptAjaxRequestJS.INTERCEPT_AJAX_REQUEST_JS_PLUGIN_SCRIPT_GROUP_NAME,
          InterceptAjaxRequestJS.INTERCEPT_AJAX_REQUEST_JS_SOURCE,
          UserScriptInjectionTime.AT_DOCUMENT_START,
          null,
          true,
          null
  );

  public static PluginScript createInterceptOnlyAsyncAjaxRequestsPluginScript(boolean onlyAsync) {
    return new PluginScript(
            InterceptAjaxRequestJS.INTERCEPT_AJAX_REQUEST_JS_PLUGIN_SCRIPT_GROUP_NAME,
            "window." + FLAG_VARIABLE_FOR_INTERCEPT_ONLY_ASYNC_AJAX_REQUESTS_JS_SOURCE + " = " + onlyAsync +";",
            UserScriptInjectionTime.AT_DOCUMENT_START,
            null,
            true,
            null
    );
  }

  public static final String INTERCEPT_AJAX_REQUEST_JS_SOURCE = "(function(ajax) {" +
          "  var w = (window.top == null || window.top === window) ? window : window.top;" +
          "  w." + FLAG_VARIABLE_FOR_SHOULD_INTERCEPT_AJAX_REQUEST_JS_SOURCE + " = true;" +
          "  var send = ajax.prototype.send;" +
          "  var open = ajax.prototype.open;" +
          "  var setRequestHeader = ajax.prototype.setRequestHeader;" +
          "  ajax.prototype._flutter_inappwebview_url = null;" +
          "  ajax.prototype._flutter_inappwebview_method = null;" +
          "  ajax.prototype._flutter_inappwebview_isAsync = null;" +
          "  ajax.prototype._flutter_inappwebview_user = null;" +
          "  ajax.prototype._flutter_inappwebview_password = null;" +
          "  ajax.prototype._flutter_inappwebview_password = null;" +
          "  ajax.prototype._flutter_inappwebview_already_onreadystatechange_wrapped = false;" +
          "  ajax.prototype._flutter_inappwebview_request_headers = {};" +
          "  function convertRequestResponse(request, callback) {" +
          "    if (request.response != null && request.responseType != null) {" +
          "      switch (request.responseType) {" +
          "        case 'arraybuffer':" +
          "          callback(new Uint8Array(request.response));" +
          "          return;" +
          "        case 'blob':" +
          "          const reader = new FileReader();" +
          "          reader.addEventListener('loadend', function() {  " +
          "            callback(new Uint8Array(reader.result));" +
          "          });" +
          "          reader.readAsArrayBuffer(blob);" +
          "          return;" +
          "        case 'document':" +
          "          callback(request.response.documentElement.outerHTML);" +
          "          return;" +
          "        case 'json':" +
          "          callback(request.response);" +
          "          return;" +
          "      };" +
          "    }" +
          "    callback(null);" +
          "  };" +
          "  ajax.prototype.open = function(method, url, isAsync, user, password) {" +
          "    isAsync = (isAsync != null) ? isAsync : true;" +
          "    this._flutter_inappwebview_url = url;" +
          "    this._flutter_inappwebview_method = method;" +
          "    this._flutter_inappwebview_isAsync = isAsync;" +
          "    this._flutter_inappwebview_user = user;" +
          "    this._flutter_inappwebview_password = password;" +
          "    this._flutter_inappwebview_request_headers = {};" +
          "    open.call(this, method, url, isAsync, user, password);" +
          "  };" +
          "  ajax.prototype.setRequestHeader = function(header, value) {" +
          "    this._flutter_inappwebview_request_headers[header] = value;" +
          "    setRequestHeader.call(this, header, value);" +
          "  };" +
          "  function handleEvent(e) {" +
          "    var self = this;" +
          "    var w = (window.top == null || window.top === window) ? window : window.top;" +
          "    if (w." + FLAG_VARIABLE_FOR_SHOULD_INTERCEPT_AJAX_REQUEST_JS_SOURCE + " == null || w." + FLAG_VARIABLE_FOR_SHOULD_INTERCEPT_AJAX_REQUEST_JS_SOURCE + " == true) {" +
          "      var headers = this.getAllResponseHeaders();" +
          "      var responseHeaders = {};" +
          "      if (headers != null) {" +
          "        var arr = headers.trim().split(/[\\r\\n]+/);" +
          "        arr.forEach(function (line) {" +
          "          var parts = line.split(': ');" +
          "          var header = parts.shift();" +
          "          var value = parts.join(': ');" +
          "          responseHeaders[header] = value;" +
          "        });" +
          "      }" +
          "      convertRequestResponse(this, function(response) {" +
          "        var ajaxRequest = {" +
          "          method: self._flutter_inappwebview_method," +
          "          url: self._flutter_inappwebview_url," +
          "          isAsync: self._flutter_inappwebview_isAsync," +
          "          user: self._flutter_inappwebview_user," +
          "          password: self._flutter_inappwebview_password," +
          "          withCredentials: self.withCredentials," +
          "          headers: self._flutter_inappwebview_request_headers," +
          "          readyState: self.readyState," +
          "          status: self.status," +
          "          responseURL: self.responseURL," +
          "          responseType: self.responseType," +
          "          response: response," +
          "          responseText: (self.responseType == 'text' || self.responseType == '') ? self.responseText : null," +
          "          responseXML: (self.responseType == 'document' && self.responseXML != null) ? self.responseXML.documentElement.outerHTML : null," +
          "          statusText: self.statusText," +
          "          responseHeaders, responseHeaders," +
          "          event: {" +
          "            type: e.type," +
          "            loaded: e.loaded," +
          "            lengthComputable: e.lengthComputable," +
          "            total: e.total" +
          "          }" +
          "        };" +
          "        window." + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + ".callHandler('onAjaxProgress', ajaxRequest).then(function(result) {" +
          "          if (result != null) {" +
          "            switch (result) {" +
          "              case 0:" +
          "                self.abort();" +
          "                return;" +
          "            };" +
          "          }" +
          "        });" +
          "      });" +
          "    }" +
          "  };" +
          "  ajax.prototype.send = function(data) {" +
          "    var self = this;" +
          "    var w = (window.top == null || window.top === window) ? window : window.top;" +
          "    var canBeIntercepted = self._flutter_inappwebview_isAsync || w." + FLAG_VARIABLE_FOR_INTERCEPT_ONLY_ASYNC_AJAX_REQUESTS_JS_SOURCE + " === false;" +
          "    if (canBeIntercepted && (w." + FLAG_VARIABLE_FOR_SHOULD_INTERCEPT_AJAX_REQUEST_JS_SOURCE + " == null || w." + FLAG_VARIABLE_FOR_SHOULD_INTERCEPT_AJAX_REQUEST_JS_SOURCE + " == true)) {" +
          "      if (!this._flutter_inappwebview_already_onreadystatechange_wrapped) {" +
          "        this._flutter_inappwebview_already_onreadystatechange_wrapped = true;" +
          "        var onreadystatechange = this.onreadystatechange;" +
          "        this.onreadystatechange = function() {" +
          "          var w = (window.top == null || window.top === window) ? window : window.top;" +
          "          if (w." + FLAG_VARIABLE_FOR_SHOULD_INTERCEPT_AJAX_REQUEST_JS_SOURCE + " == null || w." + FLAG_VARIABLE_FOR_SHOULD_INTERCEPT_AJAX_REQUEST_JS_SOURCE + " == true) {" +
          "            var headers = this.getAllResponseHeaders();" +
          "            var responseHeaders = {};" +
          "            if (headers != null) {" +
          "              var arr = headers.trim().split(/[\\r\\n]+/);" +
          "              arr.forEach(function (line) {" +
          "                var parts = line.split(': ');" +
          "                var header = parts.shift();" +
          "                var value = parts.join(': ');" +
          "                responseHeaders[header] = value;" +
          "              });" +
          "            }" +
          "            convertRequestResponse(this, function(response) {" +
          "              var ajaxRequest = {" +
          "                method: self._flutter_inappwebview_method," +
          "                url: self._flutter_inappwebview_url," +
          "                isAsync: self._flutter_inappwebview_isAsync," +
          "                user: self._flutter_inappwebview_user," +
          "                password: self._flutter_inappwebview_password," +
          "                withCredentials: self.withCredentials," +
          "                headers: self._flutter_inappwebview_request_headers," +
          "                readyState: self.readyState," +
          "                status: self.status," +
          "                responseURL: self.responseURL," +
          "                responseType: self.responseType," +
          "                response: response," +
          "                responseText: (self.responseType == 'text' || self.responseType == '') ? self.responseText : null," +
          "                responseXML: (self.responseType == 'document' && self.responseXML != null) ? self.responseXML.documentElement.outerHTML : null," +
          "                statusText: self.statusText," +
          "                responseHeaders: responseHeaders" +
          "              };" +
          "              window." + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + ".callHandler('onAjaxReadyStateChange', ajaxRequest).then(function(result) {" +
          "                if (result != null) {" +
          "                  switch (result) {" +
          "                    case 0:" +
          "                      self.abort();" +
          "                      return;" +
          "                  };" +
          "                }" +
          "                if (onreadystatechange != null) {" +
          "                  onreadystatechange();" +
          "                }" +
          "              });" +
          "            });" +
          "          } else if (onreadystatechange != null) {" +
          "            onreadystatechange();" +
          "          }" +
          "        };" +
          "      }" +
          "      this.addEventListener('loadstart', handleEvent);" +
          "      this.addEventListener('load', handleEvent);" +
          "      this.addEventListener('loadend', handleEvent);" +
          "      this.addEventListener('progress', handleEvent);" +
          "      this.addEventListener('error', handleEvent);" +
          "      this.addEventListener('abort', handleEvent);" +
          "      this.addEventListener('timeout', handleEvent);" +
          "      " + JavaScriptBridgeJS.JAVASCRIPT_UTIL_VAR_NAME + ".convertBodyRequest(data).then(function(data) {" +
          "        var ajaxRequest = {" +
          "          data: data," +
          "          method: self._flutter_inappwebview_method," +
          "          url: self._flutter_inappwebview_url," +
          "          isAsync: self._flutter_inappwebview_isAsync," +
          "          user: self._flutter_inappwebview_user," +
          "          password: self._flutter_inappwebview_password," +
          "          withCredentials: self.withCredentials," +
          "          headers: self._flutter_inappwebview_request_headers," +
          "          responseType: self.responseType" +
          "        };" +
          "        window." + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + ".callHandler('shouldInterceptAjaxRequest', ajaxRequest).then(function(result) {" +
          "          if (result != null) {" +
          "            switch (result) {" +
          "              case 0:" +
          "                self.abort();" +
          "                return;" +
          "            };" +
          "            if (result.data != null && !" + JavaScriptBridgeJS.JAVASCRIPT_UTIL_VAR_NAME + ".isString(result.data) && result.data.length > 0) {" +
          "              var bodyString = " + JavaScriptBridgeJS.JAVASCRIPT_UTIL_VAR_NAME + ".arrayBufferToString(result.data);" +
          "              if (" + JavaScriptBridgeJS.JAVASCRIPT_UTIL_VAR_NAME + ".isBodyFormData(bodyString)) {" +
          "                var formDataContentType = " + JavaScriptBridgeJS.JAVASCRIPT_UTIL_VAR_NAME + ".getFormDataContentType(bodyString);" +
          "                if (result.headers != null) {" +
          "                  result.headers['Content-Type'] = result.headers['Content-Type'] == null ? formDataContentType : result.headers['Content-Type'];" +
          "                } else {" +
          "                  result.headers = { 'Content-Type': formDataContentType };" +
          "                }" +
          "              }" +
          "            }" +
          "            if (" + JavaScriptBridgeJS.JAVASCRIPT_UTIL_VAR_NAME + ".isString(result.data) || result.data == null) {" +
          "              data = result.data;" +
          "            } else if (result.data.length > 0) {" +
          "              data = new Uint8Array(result.data);" +
          "            }" +
          "            self.withCredentials = result.withCredentials;" +
          "            if (result.responseType != null && self._flutter_inappwebview_isAsync) {" +
          "              self.responseType = result.responseType;" +
          "            };" +
          "            for (var header in result.headers) {" +
          "              var value = result.headers[header];" +
          "              var flutter_inappwebview_value = self._flutter_inappwebview_request_headers[header];" +
          "              if (flutter_inappwebview_value == null) {" +
          "                self._flutter_inappwebview_request_headers[header] = value;" +
          "              } else {" +
          "                self._flutter_inappwebview_request_headers[header] += ', ' + value;" +
          "              }" +
          "              setRequestHeader.call(self, header, value);" +
          "            };" +
          "            if ((self._flutter_inappwebview_method != result.method && result.method != null) ||" +
          "                (self._flutter_inappwebview_url != result.url && result.url != null) ||" +
          "                (self._flutter_inappwebview_isAsync != result.isAsync && result.isAsync != null) ||" +
          "                (self._flutter_inappwebview_user != result.user && result.user != null) ||" +
          "                (self._flutter_inappwebview_password != result.password && result.password != null)) {" +
          "              self.abort();" +
          "              self.open(result.method, result.url, result.isAsync, result.user, result.password);" +
          "            }" +
          "          }" +
          "          send.call(self, data);" +
          "        });" +
          "      });" +
          "    } else {" +
          "      send.call(this, data);" +
          "    }" +
          "  };" +
          "})(window.XMLHttpRequest);";
}
