package com.pichillilorenzo.flutter_inappwebview_android;

import android.net.Uri;
import androidx.webkit.ProfileStore;
import androidx.webkit.ProxyConfig;
import androidx.webkit.ProxyController;
import androidx.webkit.WebViewFeature;
import io.flutter.plugin.common.MethodCall;
import io.flutter.plugin.common.MethodChannel;

// Internal host bridge: profile creation and proxy completion precede navigation.
final class GopeedProfiles {
  private static String appliedProxy;
  private static String pendingProxy;
  private static final java.util.List<MethodChannel.Result> pending = new java.util.ArrayList<>();
  static void prepare(MethodCall call, MethodChannel.Result result) {
    if (!WebViewFeature.isFeatureSupported(WebViewFeature.MULTI_PROFILE) ||
        !WebViewFeature.isFeatureSupported(WebViewFeature.PROXY_OVERRIDE)) {
      result.error("UNAVAILABLE", "Android WebView does not support isolated profiles and proxy override", null);
      return;
    }
    String id = call.argument("gopeedProfileId");
    String proxy = call.argument("proxyUrl");
    try {
      java.util.UUID.fromString(id);
      Uri uri = Uri.parse(proxy);
      if (!"http".equals(uri.getScheme()) || !"127.0.0.1".equals(uri.getHost()) || uri.getPort() <= 0)
        throw new IllegalArgumentException("Invalid host proxy configuration");
      ProfileStore.getInstance().getOrCreateProfile(id);
      if (proxy.equals(appliedProxy)) { result.success(true); return; }
      if (pendingProxy != null) {
        if (proxy.equals(pendingProxy)) pending.add(result);
        else result.error("UNAVAILABLE", "A different WebView proxy is being configured", null);
        return;
      }
      pendingProxy = proxy;
      pending.add(result);
      ProxyConfig config = new ProxyConfig.Builder().addProxyRule(proxy).removeImplicitRules().build();
      ProxyController.getInstance().setProxyOverride(config, task -> new android.os.Handler(android.os.Looper.getMainLooper()).post(task), () -> {
        appliedProxy = proxy;
        pendingProxy = null;
        java.util.List<MethodChannel.Result> completed = new java.util.ArrayList<>(pending);
        pending.clear();
        for (MethodChannel.Result callback : completed) callback.success(true);
      });
    } catch (RuntimeException e) {
      if (pendingProxy != null && pending.contains(result)) {
        pendingProxy = null;
        java.util.List<MethodChannel.Result> failed = new java.util.ArrayList<>(pending);
        pending.clear();
        for (MethodChannel.Result callback : failed) callback.error("UNAVAILABLE", "Unable to configure WebView profile and proxy", null);
      } else result.error("UNAVAILABLE", "Unable to configure WebView profile and proxy", null);
    }
  }
}
