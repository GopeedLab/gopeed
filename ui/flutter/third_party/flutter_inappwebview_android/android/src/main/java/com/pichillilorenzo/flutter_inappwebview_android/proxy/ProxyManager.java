package com.pichillilorenzo.flutter_inappwebview_android.proxy;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.webkit.ProxyConfig;
import androidx.webkit.ProxyController;
import androidx.webkit.WebViewFeature;

import com.pichillilorenzo.flutter_inappwebview_android.InAppWebViewFlutterPlugin;
import com.pichillilorenzo.flutter_inappwebview_android.types.ChannelDelegateImpl;
import com.pichillilorenzo.flutter_inappwebview_android.types.ProxyRuleExt;

import java.util.HashMap;
import java.util.concurrent.Executor;

import io.flutter.plugin.common.MethodCall;
import io.flutter.plugin.common.MethodChannel;

public class ProxyManager extends ChannelDelegateImpl {
  protected static final String LOG_TAG = "ProxyManager";
  public static final String METHOD_CHANNEL_NAME = "com.pichillilorenzo/flutter_inappwebview_proxycontroller";

  @Nullable
  public static ProxyController proxyController;
  @Nullable
  public InAppWebViewFlutterPlugin plugin;

  public ProxyManager(@NonNull final InAppWebViewFlutterPlugin plugin) {
    super(new MethodChannel(plugin.messenger, METHOD_CHANNEL_NAME));
    this.plugin = plugin;
  }

  public static void init() {
    if (proxyController == null &&
            WebViewFeature.isFeatureSupported(WebViewFeature.PROXY_OVERRIDE)) {
      proxyController = ProxyController.getInstance();
    }
  }

  @Override
  public void onMethodCall(@NonNull MethodCall call, @NonNull MethodChannel.Result result) {
    init();

    switch (call.method) {
      case "setProxyOverride":
        if (proxyController != null) {
          HashMap<String, Object> settingsMap = (HashMap<String, Object>) call.argument("settings");
          ProxySettings settings = new ProxySettings();
          if (settingsMap != null) {
            settings.parse(settingsMap);
          }
          setProxyOverride(settings, result);
        } else {
          result.success(false);
        }
        break;
      case "clearProxyOverride":
        if (proxyController != null ) {
          clearProxyOverride(result);
        } else {
          result.success(false);
        }
        break;
      default:
        result.notImplemented();
    }
  }
  
  private void setProxyOverride(ProxySettings settings, final MethodChannel.Result result) {
    if (proxyController != null) {
      ProxyConfig.Builder proxyConfigBuilder = new ProxyConfig.Builder();
      for (String bypassRule : settings.bypassRules) {
        proxyConfigBuilder.addBypassRule(bypassRule);
      }
      for (String direct : settings.directs) {
        proxyConfigBuilder.addDirect(direct);
      }
      for (ProxyRuleExt proxyRule : settings.proxyRules) {
        if (proxyRule.getSchemeFilter() != null) {
          proxyConfigBuilder.addProxyRule(proxyRule.getUrl(), proxyRule.getSchemeFilter());
        } else {
          proxyConfigBuilder.addProxyRule(proxyRule.getUrl());
        }
      }
      if (settings.bypassSimpleHostnames != null && settings.bypassSimpleHostnames) {
        proxyConfigBuilder.bypassSimpleHostnames();
      }
      if (settings.removeImplicitRules != null && settings.removeImplicitRules) {
        proxyConfigBuilder.removeImplicitRules();
      }
      if (settings.reverseBypassEnabled != null && WebViewFeature.isFeatureSupported(WebViewFeature.PROXY_OVERRIDE_REVERSE_BYPASS)) {
        proxyConfigBuilder.setReverseBypassEnabled(settings.reverseBypassEnabled);
      }
      proxyController.setProxyOverride(proxyConfigBuilder.build(), new Executor() {
        @Override
        public void execute(Runnable command) {
          command.run();
        }
      }, new Runnable() {
        @Override
        public void run() {
          result.success(true);
        }
      });
    }
  }

  private void clearProxyOverride(final MethodChannel.Result result) {
    if (proxyController != null) {
      proxyController.clearProxyOverride(new Executor() {
        @Override
        public void execute(Runnable command) {
          command.run();
        }
      }, new Runnable() {
        @Override
        public void run() {
          result.success(true);
        }
      });
    }
  }

  @Override
  public void dispose() {
    super.dispose();
    plugin = null;
  }
}
