package com.pichillilorenzo.flutter_inappwebview_android.tracing;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.webkit.TracingConfig;
import androidx.webkit.TracingController;
import androidx.webkit.WebViewFeature;

import com.pichillilorenzo.flutter_inappwebview_android.types.ChannelDelegateImpl;

import java.io.FileNotFoundException;
import java.io.FileOutputStream;
import java.util.Map;
import java.util.concurrent.Executors;

import io.flutter.plugin.common.MethodCall;
import io.flutter.plugin.common.MethodChannel;

public class TracingControllerChannelDelegate extends ChannelDelegateImpl {
  @Nullable
  private TracingControllerManager tracingControllerManager;

  public TracingControllerChannelDelegate(@NonNull TracingControllerManager tracingControllerManager, @NonNull MethodChannel channel) {
    super(channel);
    this.tracingControllerManager = tracingControllerManager;
  }

  @Override
  public void onMethodCall(@NonNull MethodCall call, @NonNull MethodChannel.Result result) {
    TracingControllerManager.init();
    TracingController tracingController = TracingControllerManager.tracingController;

    switch (call.method) {
      case "isTracing":
        if (tracingController != null) {
          result.success(tracingController.isTracing());
        } else {
          result.success(false);
        }
        break;
      case "start":
        if (tracingController != null && WebViewFeature.isFeatureSupported(WebViewFeature.TRACING_CONTROLLER_BASIC_USAGE)) {
          Map<String, Object> settingsMap = (Map<String, Object>) call.argument("settings");
          TracingSettings settings = new TracingSettings();
          settings.parse(settingsMap);
          TracingConfig config = TracingControllerManager.buildTracingConfig(settings);
          tracingController.start(config);
          result.success(true);
        } else {
          result.success(false);
        }
        break;
      case "stop":
        if (tracingController != null && WebViewFeature.isFeatureSupported(WebViewFeature.TRACING_CONTROLLER_BASIC_USAGE)) {
          String filePath = (String) call.argument("filePath");
          try {
            result.success(tracingController.stop(
                    filePath != null ? new FileOutputStream(filePath) : null,
                    Executors.newSingleThreadExecutor()));
          } catch (FileNotFoundException e) {
            e.printStackTrace();
            result.success(false);
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
    tracingControllerManager = null;
  }
}
