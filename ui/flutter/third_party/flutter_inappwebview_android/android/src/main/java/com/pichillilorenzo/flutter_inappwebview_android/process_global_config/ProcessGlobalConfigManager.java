package com.pichillilorenzo.flutter_inappwebview_android.process_global_config;

import android.util.Log;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.webkit.ProcessGlobalConfig;

import com.pichillilorenzo.flutter_inappwebview_android.InAppWebViewFlutterPlugin;
import com.pichillilorenzo.flutter_inappwebview_android.types.ChannelDelegateImpl;

import java.util.Map;

import io.flutter.plugin.common.MethodCall;
import io.flutter.plugin.common.MethodChannel;

public class ProcessGlobalConfigManager extends ChannelDelegateImpl {
  protected static final String LOG_TAG = "ProcessGlobalConfigM";
  public static final String METHOD_CHANNEL_NAME = "com.pichillilorenzo/flutter_inappwebview_processglobalconfig";

  @Nullable
  public InAppWebViewFlutterPlugin plugin;

  public ProcessGlobalConfigManager(@NonNull final InAppWebViewFlutterPlugin plugin) {
    super(new MethodChannel(plugin.messenger, METHOD_CHANNEL_NAME));
    this.plugin = plugin;
  }

  @Override
  public void onMethodCall(@NonNull MethodCall call, @NonNull MethodChannel.Result result) {
    switch (call.method) {
      case "apply":
        if (plugin != null && plugin.activity != null) {
          ProcessGlobalConfigSettings settings = (new ProcessGlobalConfigSettings())
                  .parse((Map<String, Object>) call.argument("settings"));
          try {
            ProcessGlobalConfig.apply(settings.toProcessGlobalConfig(plugin.activity));
            result.success(true);
          } catch (Exception e) {
            result.error(LOG_TAG, "", e);
          }
        } else {
          result.success(false);
        }
        break;
      default:
        result.notImplemented();
    }
  }

  @Override
  public void dispose() {
    super.dispose();
    plugin = null;
  }
}
