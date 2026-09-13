package com.pichillilorenzo.flutter_inappwebview_android.webview.web_message;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import com.pichillilorenzo.flutter_inappwebview_android.types.ChannelDelegateImpl;
import com.pichillilorenzo.flutter_inappwebview_android.types.WebMessageCompatExt;
import com.pichillilorenzo.flutter_inappwebview_android.webview.in_app_webview.InAppWebView;

import java.util.HashMap;
import java.util.Map;

import io.flutter.plugin.common.MethodCall;
import io.flutter.plugin.common.MethodChannel;

public class WebMessageChannelChannelDelegate extends ChannelDelegateImpl {
  @Nullable
  private WebMessageChannel webMessageChannel;

  public WebMessageChannelChannelDelegate(@NonNull WebMessageChannel webMessageChannel, @NonNull MethodChannel channel) {
    super(channel);
    this.webMessageChannel = webMessageChannel;
  }

  @Override
  public void onMethodCall(@NonNull MethodCall call, @NonNull MethodChannel.Result result) {
    switch (call.method) {
      case "setWebMessageCallback":
        if (webMessageChannel != null && webMessageChannel.webView instanceof InAppWebView) {
          final Integer index = (Integer) call.argument("index");
          webMessageChannel.setWebMessageCallbackForInAppWebView(index, result);
        } else {
          result.success(false);
        }
        break;
      case "postMessage":
        if (webMessageChannel != null && webMessageChannel.webView instanceof InAppWebView) {
          final Integer index = (Integer) call.argument("index");
          WebMessageCompatExt message = WebMessageCompatExt.fromMap((Map<String, Object>) call.argument("message"));
          webMessageChannel.postMessageForInAppWebView(index, message, result);
        } else {
          result.success(false);
        }
        break;
      case "close":
        if (webMessageChannel != null && webMessageChannel.webView instanceof InAppWebView) {
          Integer index = (Integer) call.argument("index");
          webMessageChannel.closeForInAppWebView(index, result);
        } else {
          result.success(false);
        }
        break;
      default:
        result.notImplemented();
    }
  }

  public void onMessage(int index, @Nullable WebMessageCompatExt message) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("index", index);
    obj.put("message", message != null ? message.toMap() : null);
    channel.invokeMethod("onMessage", obj);
  }

  @Override
  public void dispose() {
    super.dispose();
    webMessageChannel = null;
  }
}
