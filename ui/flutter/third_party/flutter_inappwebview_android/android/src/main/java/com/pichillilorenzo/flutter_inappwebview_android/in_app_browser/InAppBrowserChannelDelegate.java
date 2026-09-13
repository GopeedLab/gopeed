package com.pichillilorenzo.flutter_inappwebview_android.in_app_browser;

import androidx.annotation.NonNull;

import com.pichillilorenzo.flutter_inappwebview_android.types.ChannelDelegateImpl;
import com.pichillilorenzo.flutter_inappwebview_android.types.InAppBrowserMenuItem;

import java.util.HashMap;
import java.util.Map;

import io.flutter.plugin.common.MethodChannel;

public class InAppBrowserChannelDelegate extends ChannelDelegateImpl {
  public InAppBrowserChannelDelegate(@NonNull MethodChannel channel) {
    super(channel);
  }

  public void onBrowserCreated() {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    channel.invokeMethod("onBrowserCreated", obj);
  }

  public void onMenuItemClicked(InAppBrowserMenuItem menuItem) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("id", menuItem.getId());
    channel.invokeMethod("onMenuItemClicked", obj);
  }

  public void onExit() {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    channel.invokeMethod("onExit", obj);
  }
}
