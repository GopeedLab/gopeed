package com.pichillilorenzo.flutter_inappwebview_android;

import android.net.Uri;
import android.content.Context;
import android.content.SharedPreferences;
import androidx.webkit.Profile;
import androidx.webkit.WebStorageCompat;
import java.util.HashSet;
import java.util.Set;
import androidx.webkit.ProfileStore;
import androidx.webkit.ProxyConfig;
import androidx.webkit.ProxyController;
import androidx.webkit.WebViewFeature;
import io.flutter.plugin.common.MethodCall;
import io.flutter.plugin.common.MethodChannel;

// Internal host bridge: profile creation and proxy completion precede navigation.
final class GopeedProfiles {
  private static SharedPreferences preferences;
  private static final Set<String> cleared = new HashSet<>();
  private static final String DELETIONS = "pendingDeletions";

  static void initialize(Context context) {
    if (preferences != null) return;
    preferences = context.getSharedPreferences("gopeed_webview_profiles", Context.MODE_PRIVATE);
    if (!WebViewFeature.isFeatureSupported(WebViewFeature.MULTI_PROFILE)) return;
    for (String id : deletions()) {
      try {
        // Do this before getProfile/getOrCreateProfile load any profile.
        ProfileStore.getInstance().deleteProfile(id);
        setPending(id, false);
      } catch (RuntimeException ignored) {
        // Keep the durable entry and reject use until removal succeeds.
      }
    }
  }

  private static Set<String> deletions() {
    return new HashSet<>(preferences.getStringSet(DELETIONS, new HashSet<>()));
  }

  private static void setPending(String id, boolean pending) {
    Set<String> ids = deletions();
    if (pending) ids.add(id); else ids.remove(id);
    if (!preferences.edit().putStringSet(DELETIONS, ids).commit())
      throw new IllegalStateException("Unable to persist WebView profile removal");
  }

  static void remove(MethodCall call, MethodChannel.Result result) {
    if (!WebViewFeature.isFeatureSupported(WebViewFeature.MULTI_PROFILE)) {
      result.error("UNAVAILABLE", "Android WebView profile removal is unavailable", null);
      return;
    }
    String id = call.argument("gopeedProfileId");
    try {
      java.util.UUID.fromString(id);
      setPending(id, true);
      try {
        ProfileStore.getInstance().deleteProfile(id);
        setPending(id, false);
        cleared.remove(id);
        result.success(true);
        return;
      } catch (IllegalStateException loadedProfile) {
        // Android forbids deleting a profile loaded in this process, even
        // after every WebView has been destroyed. Clear all its data now;
        // remove the empty profile at the next process start.
      }
      if (!WebViewFeature.isFeatureSupported(WebViewFeature.DELETE_BROWSING_DATA)) {
        result.error("PROFILE_REMOVE_FAILED", "Restart Gopeed to finish removing WebView profile data", null);
        return;
      }
      Profile profile = ProfileStore.getInstance().getProfile(id);
      if (profile == null) {
        setPending(id, false);
        result.success(true);
        return;
      }
      profile.getGeolocationPermissions().clearAll();
      WebStorageCompat.deleteBrowsingData(profile.getWebStorage(), () -> {
        cleared.add(id);
        result.success(true);
      });
    } catch (RuntimeException e) {
      result.error("PROFILE_REMOVE_FAILED", "Unable to remove WebView profile data", null);
    }
  }

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
      if (deletions().contains(id)) {
        if (!cleared.contains(id)) throw new IllegalStateException("Profile removal requires restart");
        // Reinstallation may reuse the now-empty profile. Do not delete its
        // new data on the next start.
        setPending(id, false);
        cleared.remove(id);
      }
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
