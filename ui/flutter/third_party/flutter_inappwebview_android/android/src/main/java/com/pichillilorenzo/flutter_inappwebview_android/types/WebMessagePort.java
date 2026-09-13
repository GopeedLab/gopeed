package com.pichillilorenzo.flutter_inappwebview_android.types;

import android.text.TextUtils;
import android.webkit.ValueCallback;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import com.pichillilorenzo.flutter_inappwebview_android.Util;
import com.pichillilorenzo.flutter_inappwebview_android.webview.InAppWebViewInterface;
import com.pichillilorenzo.flutter_inappwebview_android.webview.web_message.WebMessageChannel;
import com.pichillilorenzo.flutter_inappwebview_android.plugin_scripts_js.JavaScriptBridgeJS;

import java.util.ArrayList;
import java.util.List;

public class WebMessagePort {
  public String name;
  @Nullable
  public WebMessageChannel webMessageChannel;
  public boolean isClosed = false;
  public boolean isTransferred = false;
  public boolean isStarted = false;

  public WebMessagePort(String name, @NonNull WebMessageChannel webMessageChannel) {
    this.name = name;
    this.webMessageChannel = webMessageChannel;
  }

  public void setWebMessageCallback(final ValueCallback<Void> callback) throws Exception {
    if (isClosed || isTransferred) {
      throw new Exception("Port is already closed or transferred");
    }
    this.isStarted = true;
    InAppWebViewInterface webView = webMessageChannel != null && webMessageChannel.webView != null ? webMessageChannel.webView : null;
    if (webView != null) {
      int index = name.equals("port1") ? 0 : 1;
      webView.evaluateJavascript("(function() {" +
              "  var webMessageChannel = " + JavaScriptBridgeJS.WEB_MESSAGE_CHANNELS_VARIABLE_NAME + "['" + webMessageChannel.id + "'];" +
              "  if (webMessageChannel != null) {" +
              "      webMessageChannel." + this.name + ".onmessage = function (event) {" +
              "          window." + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + ".callHandler('onWebMessagePortMessageReceived', {" +
              "              'webMessageChannelId': '" + webMessageChannel.id + "'," +
              "              'index': " + index + "," +
              "              'message': event.data" +
              "          });" +
              "      }" +
              "  }" +
              "})();", null, new ValueCallback<String>() {
        @Override
        public void onReceiveValue(String value) {
          if (callback != null) {
            callback.onReceiveValue(null);
          }
        }
      });
    } else {
      if (callback != null) {
        callback.onReceiveValue(null);
      }
    }
  }

  public void postMessage(WebMessage message, final ValueCallback<Void> callback) throws Exception {
    if (isClosed || isTransferred) {
      throw new Exception("Port is already closed or transferred");
    }
    InAppWebViewInterface webView = webMessageChannel != null && webMessageChannel.webView != null ? webMessageChannel.webView : null;
    if (webView != null) {
      String portsString = "null";
      List<WebMessagePort> ports = message.ports;
      if (ports != null) {
        List<String> portArrayString = new ArrayList<>();
        for (WebMessagePort port : ports) {
          if (port == this) {
            throw new Exception("Source port cannot be transferred");
          }
          if (port.isStarted) {
            throw new Exception("Port is already started");
          }
          if (port.isClosed || port.isTransferred) {
            throw new Exception("Port is already closed or transferred");
          }
          port.isTransferred = true;
          portArrayString.add(JavaScriptBridgeJS.WEB_MESSAGE_CHANNELS_VARIABLE_NAME + "['" + webMessageChannel.id + "']." + port.name);
        }
        portsString = "[" + TextUtils.join(", ", portArrayString) + "]";
      }
      String data = message.data != null ? Util.replaceAll(message.data, "\'", "\\'") : "null";
      String source = "(function() {" +
              "  var webMessageChannel = " + JavaScriptBridgeJS.WEB_MESSAGE_CHANNELS_VARIABLE_NAME + "['" + webMessageChannel.id + "'];" +
              "  if (webMessageChannel != null) {" +
              "      webMessageChannel." + this.name + ".postMessage('" + data + "', " + portsString + ");" +
              "  }" +
              "})();";
      webView.evaluateJavascript(source, null, new ValueCallback<String>() {
        @Override
        public void onReceiveValue(String value) {
          callback.onReceiveValue(null);
        }
      });
    } else {
      callback.onReceiveValue(null);
    }
    message.dispose();
  }

  public void close(final ValueCallback<Void> callback) throws Exception {
    if (isTransferred) {
      throw new Exception("Port is already transferred");
    }
    isClosed = true;
    InAppWebViewInterface webView = webMessageChannel != null && webMessageChannel.webView != null ? webMessageChannel.webView : null;
    if (webView != null) {
      String source = "(function() {" +
              "  var webMessageChannel = " + JavaScriptBridgeJS.WEB_MESSAGE_CHANNELS_VARIABLE_NAME + "['" + webMessageChannel.id + "'];" +
              "  if (webMessageChannel != null) {" +
              "      webMessageChannel." + this.name + ".close();" +
              "  }" +
              "})();";
      webView.evaluateJavascript(source, null, new ValueCallback<String>() {
        @Override
        public void onReceiveValue(String value) {
          callback.onReceiveValue(null);
        }
      });
    } else {
      callback.onReceiveValue(null);
    }
  }

  public void dispose() {
    isClosed = true;
    webMessageChannel = null;
  }
}
