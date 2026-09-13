package com.pichillilorenzo.flutter_inappwebview_android;

import android.app.Activity;
import android.content.Context;
import android.os.Build;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import com.pichillilorenzo.flutter_inappwebview_android.chrome_custom_tabs.ChromeSafariBrowserManager;
import com.pichillilorenzo.flutter_inappwebview_android.chrome_custom_tabs.NoHistoryCustomTabsActivityCallbacks;
import com.pichillilorenzo.flutter_inappwebview_android.credential_database.CredentialDatabaseHandler;
import com.pichillilorenzo.flutter_inappwebview_android.headless_in_app_webview.HeadlessInAppWebViewManager;
import com.pichillilorenzo.flutter_inappwebview_android.in_app_browser.InAppBrowserManager;
import com.pichillilorenzo.flutter_inappwebview_android.print_job.PrintJobManager;
import com.pichillilorenzo.flutter_inappwebview_android.process_global_config.ProcessGlobalConfigManager;
import com.pichillilorenzo.flutter_inappwebview_android.proxy.ProxyManager;
import com.pichillilorenzo.flutter_inappwebview_android.service_worker.ServiceWorkerManager;
import com.pichillilorenzo.flutter_inappwebview_android.tracing.TracingControllerManager;
import com.pichillilorenzo.flutter_inappwebview_android.webview.FlutterWebViewFactory;
import com.pichillilorenzo.flutter_inappwebview_android.webview.InAppWebViewManager;

import io.flutter.embedding.engine.plugins.FlutterPlugin;
import io.flutter.embedding.engine.plugins.activity.ActivityAware;
import io.flutter.embedding.engine.plugins.activity.ActivityPluginBinding;
import io.flutter.plugin.common.BinaryMessenger;
import io.flutter.plugin.platform.PlatformViewRegistry;
import io.flutter.embedding.android.FlutterView;

public class InAppWebViewFlutterPlugin implements FlutterPlugin, ActivityAware {

  protected static final String LOG_TAG = "InAppWebViewFlutterPL";

  @Nullable
  public PlatformUtil platformUtil;
  @Nullable
  public InAppBrowserManager inAppBrowserManager;
  @Nullable
  public HeadlessInAppWebViewManager headlessInAppWebViewManager;
  @Nullable
  public ChromeSafariBrowserManager chromeSafariBrowserManager;
  @Nullable
  public NoHistoryCustomTabsActivityCallbacks noHistoryCustomTabsActivityCallbacks;
  @Nullable
  public InAppWebViewManager inAppWebViewManager;
  @Nullable
  public MyCookieManager myCookieManager;
  @Nullable
  public CredentialDatabaseHandler credentialDatabaseHandler;
  @Nullable
  public MyWebStorage myWebStorage;
  @Nullable
  public ServiceWorkerManager serviceWorkerManager;
  @Nullable
  public WebViewFeatureManager webViewFeatureManager;
  @Nullable
  public ProxyManager proxyManager;
  @Nullable
  public PrintJobManager printJobManager;
  @Nullable
  public TracingControllerManager tracingControllerManager;
  @Nullable
  public ProcessGlobalConfigManager processGlobalConfigManager;
  public FlutterWebViewFactory flutterWebViewFactory;
  public Context applicationContext;
  public BinaryMessenger messenger;
  public FlutterPlugin.FlutterAssets flutterAssets;
  @Nullable
  public ActivityPluginBinding activityPluginBinding;
  @Nullable
  public Activity activity;
  public FlutterView flutterView;

  public InAppWebViewFlutterPlugin() {}

  @Override
  public void onAttachedToEngine(FlutterPluginBinding binding) {
    this.flutterAssets = binding.getFlutterAssets();

    // Shared.activity could be null or not.
    // It depends on who is called first between onAttachedToEngine event and onAttachedToActivity event.
    //
    // See https://github.com/pichillilorenzo/flutter_inappwebview/issues/390#issuecomment-647039084
    onAttachedToEngine(
            binding.getApplicationContext(), binding.getBinaryMessenger(), this.activity, binding.getPlatformViewRegistry(), null);
  }

  private void onAttachedToEngine(Context applicationContext, BinaryMessenger messenger, Activity activity, PlatformViewRegistry platformViewRegistry, FlutterView flutterView) {
    this.applicationContext = applicationContext;
    this.activity = activity;
    this.messenger = messenger;
    this.flutterView = flutterView;

    inAppBrowserManager = new InAppBrowserManager(this);
    headlessInAppWebViewManager = new HeadlessInAppWebViewManager(this);
    chromeSafariBrowserManager = new ChromeSafariBrowserManager(this);
    noHistoryCustomTabsActivityCallbacks = new NoHistoryCustomTabsActivityCallbacks(this);
    flutterWebViewFactory = new FlutterWebViewFactory(this);
    platformViewRegistry.registerViewFactory(
            FlutterWebViewFactory.VIEW_TYPE_ID, flutterWebViewFactory);

    platformUtil = new PlatformUtil(this);
    inAppWebViewManager = new InAppWebViewManager(this);
    myCookieManager = new MyCookieManager(this);
    myWebStorage = new MyWebStorage(this);
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.N) {
      serviceWorkerManager = new ServiceWorkerManager(this);
    }
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
      credentialDatabaseHandler = new CredentialDatabaseHandler(this);
    }
    webViewFeatureManager = new WebViewFeatureManager(this);
    proxyManager = new ProxyManager(this);
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.KITKAT) {
      printJobManager = new PrintJobManager(this);
    }
    tracingControllerManager = new TracingControllerManager(this);
    processGlobalConfigManager = new ProcessGlobalConfigManager(this);
  }

  @Override
  public void onDetachedFromEngine(@NonNull FlutterPluginBinding binding) {
    if (platformUtil != null) {
      platformUtil.dispose();
      platformUtil = null;
    }
    if (inAppBrowserManager != null) {
      inAppBrowserManager.dispose();
      inAppBrowserManager = null;
    }
    if (headlessInAppWebViewManager != null) {
      headlessInAppWebViewManager.dispose();
      headlessInAppWebViewManager = null;
    }
    if (chromeSafariBrowserManager != null) {
      chromeSafariBrowserManager.dispose();
      chromeSafariBrowserManager = null;
    }
    if (noHistoryCustomTabsActivityCallbacks != null) {
      noHistoryCustomTabsActivityCallbacks.dispose();
      noHistoryCustomTabsActivityCallbacks = null;
    }
    if (myCookieManager != null) {
      myCookieManager.dispose();
      myCookieManager = null;
    }
    if (myWebStorage != null) {
      myWebStorage.dispose();
      myWebStorage = null;
    }
    if (credentialDatabaseHandler != null && Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
      credentialDatabaseHandler.dispose();
      credentialDatabaseHandler = null;
    }
    if (inAppWebViewManager != null) {
      inAppWebViewManager.dispose();
      inAppWebViewManager = null;
    }
    if (serviceWorkerManager != null && Build.VERSION.SDK_INT >= Build.VERSION_CODES.N) {
      serviceWorkerManager.dispose();
      serviceWorkerManager = null;
    }
    if (webViewFeatureManager != null) {
      webViewFeatureManager.dispose();
      webViewFeatureManager = null;
    }
    if (proxyManager != null) {
      proxyManager.dispose();
      proxyManager = null;
    }
    if (printJobManager != null && Build.VERSION.SDK_INT >= Build.VERSION_CODES.KITKAT) {
      printJobManager.dispose();
      printJobManager = null;
    }
    if (tracingControllerManager != null) {
      tracingControllerManager.dispose();
      tracingControllerManager = null;
    }
    if (processGlobalConfigManager != null) {
      processGlobalConfigManager.dispose();
      processGlobalConfigManager = null;
    }
  }

  @Override
  public void onAttachedToActivity(ActivityPluginBinding activityPluginBinding) {
    this.activityPluginBinding = activityPluginBinding;
    this.activity = activityPluginBinding.getActivity();

    if (noHistoryCustomTabsActivityCallbacks != null) {
      this.activity.getApplication().registerActivityLifecycleCallbacks(noHistoryCustomTabsActivityCallbacks.activityLifecycleCallbacks);
    }
  }

  @Override
  public void onDetachedFromActivityForConfigChanges() {
    if (activity != null && noHistoryCustomTabsActivityCallbacks != null) {
      this.activity.getApplication().unregisterActivityLifecycleCallbacks(noHistoryCustomTabsActivityCallbacks.activityLifecycleCallbacks);
    }

    activityPluginBinding = null;
    activity = null;
  }

  @Override
  public void onReattachedToActivityForConfigChanges(ActivityPluginBinding activityPluginBinding) {
    this.activityPluginBinding = activityPluginBinding;
    this.activity = activityPluginBinding.getActivity();

    if (noHistoryCustomTabsActivityCallbacks != null) {
      this.activity.getApplication().registerActivityLifecycleCallbacks(noHistoryCustomTabsActivityCallbacks.activityLifecycleCallbacks);
    }
  }

  @Override
  public void onDetachedFromActivity() {
    if (activity != null && noHistoryCustomTabsActivityCallbacks != null) {
      this.activity.getApplication().unregisterActivityLifecycleCallbacks(noHistoryCustomTabsActivityCallbacks.activityLifecycleCallbacks);
    }

    activityPluginBinding = null;
    activity = null;
  }
}
