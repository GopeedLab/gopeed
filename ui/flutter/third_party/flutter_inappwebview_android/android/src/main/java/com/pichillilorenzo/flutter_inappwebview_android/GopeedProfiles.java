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
      ProxyConfig config = new ProxyConfig.Builder().addProxyRule(proxy).removeImplicitRules().build();
      ProxyController.getInstance().setProxyOverride(config, Runnable::run, () -> result.success(true));
    } catch (RuntimeException e) {
      result.error("UNAVAILABLE", "Unable to configure WebView profile and proxy", null);
    }
  }
}
