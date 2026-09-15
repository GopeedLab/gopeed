package com.pichillilorenzo.flutter_inappwebview_android.webview.web_message;

import android.webkit.ValueCallback;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.webkit.WebMessageCompat;
import androidx.webkit.WebMessagePortCompat;
import androidx.webkit.WebViewCompat;
import androidx.webkit.WebViewFeature;

import com.pichillilorenzo.flutter_inappwebview_android.plugin_scripts_js.JavaScriptBridgeJS;
import com.pichillilorenzo.flutter_inappwebview_android.types.Disposable;
import com.pichillilorenzo.flutter_inappwebview_android.types.WebMessageCompatExt;
import com.pichillilorenzo.flutter_inappwebview_android.types.WebMessagePortCompatExt;
import com.pichillilorenzo.flutter_inappwebview_android.webview.InAppWebViewInterface;
import com.pichillilorenzo.flutter_inappwebview_android.types.WebMessagePort;
import com.pichillilorenzo.flutter_inappwebview_android.webview.in_app_webview.InAppWebView;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import io.flutter.plugin.common.MethodChannel;

public class WebMessageChannel implements Disposable {
  protected static final String LOG_TAG = "WebMessageChannel";
  public static final String METHOD_CHANNEL_NAME_PREFIX = "com.pichillilorenzo/flutter_inappwebview_web_message_channel_";
  
  @NonNull
  public String id;
  @Nullable
  public WebMessageChannelChannelDelegate channelDelegate;
  public final List<WebMessagePortCompat> compatPorts;
  public final List<WebMessagePort> ports;
  @Nullable
  public InAppWebViewInterface webView;

  public WebMessageChannel(@NonNull String id, @NonNull InAppWebViewInterface webView) {
    this.id = id;
    final MethodChannel channel = new MethodChannel(webView.getPlugin().messenger, METHOD_CHANNEL_NAME_PREFIX + id);
    this.channelDelegate = new WebMessageChannelChannelDelegate(this, channel);
    if (webView instanceof InAppWebView) {
      this.compatPorts = new ArrayList<>(Arrays.asList(WebViewCompat.createWebMessageChannel((InAppWebView) webView)));
      this.ports = new ArrayList<>();
    } else {
      this.ports = Arrays.asList(new WebMessagePort("port1", this), new WebMessagePort("port2", this));
      this.compatPorts = new ArrayList<>();
    }
    this.webView = webView;
  }

  public void initJsInstance(InAppWebViewInterface webView, final ValueCallback<WebMessageChannel> callback) {
    if (webView != null) {
      final WebMessageChannel webMessageChannel = this;
      webView.evaluateJavascript("(function() {" +
              JavaScriptBridgeJS.WEB_MESSAGE_CHANNELS_VARIABLE_NAME + "['" + webMessageChannel.id + "'] = new MessageChannel();" +
              "})();", null, new ValueCallback<String>() {
        @Override
        public void onReceiveValue(String value) {
          callback.onReceiveValue(webMessageChannel);
        }
      });
    } else {
      callback.onReceiveValue(this);
    }
  }
  
  public void setWebMessageCallbackForInAppWebView(final int index, @NonNull MethodChannel.Result result) {
    if (webView != null && compatPorts.size() > 0 &&
            WebViewFeature.isFeatureSupported(WebViewFeature.WEB_MESSAGE_PORT_SET_MESSAGE_CALLBACK)) {
      final WebMessagePortCompat webMessagePort = compatPorts.get(index);
      final WebMessageChannel webMessageChannel = this;
      try {
        webMessagePort.setWebMessageCallback(new WebMessagePortCompat.WebMessageCallbackCompat() {
          @Override
          public void onMessage(@NonNull WebMessagePortCompat port, @Nullable WebMessageCompat message) {
            super.onMessage(port, message);
            webMessageChannel.onMessage(index, message != null ? WebMessageCompatExt.fromMapWebMessageCompat(message) : null);
          }
        });
        result.success(true);
      } catch (Exception e) {
        result.error(LOG_TAG, e.getMessage(), null);
      }
    } else {
      result.success(true);
    }
  }

  public void postMessageForInAppWebView(final Integer index, @NonNull WebMessageCompatExt message, @NonNull MethodChannel.Result result) {
    if (webView != null && compatPorts.size() > 0 &&
            WebViewFeature.isFeatureSupported(WebViewFeature.WEB_MESSAGE_PORT_POST_MESSAGE)) {
      WebMessagePortCompat port = compatPorts.get(index);
      List<WebMessagePortCompat> webMessagePorts = new ArrayList<>();
      List<WebMessagePortCompatExt> portsExt = message.getPorts();
      if (portsExt != null) {
        for (WebMessagePortCompatExt portExt : portsExt) {
          WebMessageChannel webMessageChannel = webView.getWebMessageChannels().get(portExt.getWebMessageChannelId());
          if (webMessageChannel != null) {
            webMessagePorts.add(webMessageChannel.compatPorts.get(portExt.getIndex()));
          }
        }
      }
      Object data = message.getData();
      try {
        if (WebViewFeature.isFeatureSupported(WebViewFeature.WEB_MESSAGE_ARRAY_BUFFER) && data != null &&
                message.getType() == WebMessageCompat.TYPE_ARRAY_BUFFER) {
          port.postMessage(new WebMessageCompat((byte[]) data, webMessagePorts.toArray(new WebMessagePortCompat[0])));
        } else {
          port.postMessage(new WebMessageCompat(data != null ? data.toString() : null, webMessagePorts.toArray(new WebMessagePortCompat[0])));
        }
        result.success(true);
      } catch (Exception e) {
        result.error(LOG_TAG, e.getMessage(), null);
      }
    } else {
      result.success(true);
    }
  }

  public void closeForInAppWebView(Integer index, @NonNull MethodChannel.Result result) {
    if (webView != null && compatPorts.size() > 0 &&
            WebViewFeature.isFeatureSupported(WebViewFeature.WEB_MESSAGE_PORT_CLOSE)) {
      WebMessagePortCompat port = compatPorts.get(index);
      try {
        port.close();
        result.success(true);
      } catch (Exception e) {
        result.error(LOG_TAG, e.getMessage(), null);
      }
    } else {
      result.success(true);
    }
  }

  public void onMessage(int index, @Nullable WebMessageCompatExt message) {
    if (channelDelegate != null) {
      channelDelegate.onMessage(index, message);
    }
  }

  public Map<String, Object> toMap() {
    Map<String, Object> webMessageChannelMap = new HashMap<>();
    webMessageChannelMap.put("id", id);
    return webMessageChannelMap;
  }

  public void dispose() {
    if (WebViewFeature.isFeatureSupported(WebViewFeature.WEB_MESSAGE_PORT_CLOSE)) {
      for (WebMessagePortCompat port : compatPorts) {
        try {
          port.close();
        } catch (Exception ignored) {}
      }
    }
    if (channelDelegate != null) {
      channelDelegate.dispose();
      channelDelegate = null;
    }
    compatPorts.clear();
    webView = null;
  }
}
