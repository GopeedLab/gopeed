package com.pichillilorenzo.flutter_inappwebview_android.webview.in_app_webview;

import static android.content.Context.INPUT_METHOD_SERVICE;
import static com.pichillilorenzo.flutter_inappwebview_android.types.PreferredContentModeOptionType.fromValue;

import android.animation.ObjectAnimator;
import android.animation.PropertyValuesHolder;
import android.annotation.SuppressLint;
import android.annotation.TargetApi;
import android.content.Context;
import android.content.pm.PackageInfo;
import android.graphics.Bitmap;
import android.graphics.Canvas;
import android.graphics.Color;
import android.graphics.Point;
import android.graphics.drawable.ColorDrawable;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.os.Message;
import android.print.PrintAttributes;
import android.print.PrintDocumentAdapter;
import android.print.PrintManager;
import android.text.TextUtils;
import android.util.AttributeSet;
import android.util.Log;
import android.view.ActionMode;
import android.view.ContextMenu;
import android.view.GestureDetector;
import android.view.LayoutInflater;
import android.view.Menu;
import android.view.MenuItem;
import android.view.MotionEvent;
import android.view.View;
import android.view.ViewGroup;
import android.view.ViewParent;
import android.view.ViewTreeObserver;
import android.view.inputmethod.EditorInfo;
import android.view.inputmethod.InputConnection;
import android.view.inputmethod.InputMethodManager;
import android.webkit.CookieManager;
import android.webkit.DownloadListener;
import android.webkit.URLUtil;
import android.webkit.ValueCallback;
import android.webkit.WebBackForwardList;
import android.webkit.WebChromeClient;
import android.webkit.WebHistoryItem;
import android.webkit.WebSettings;
import android.webkit.WebStorage;
import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.widget.HorizontalScrollView;
import android.widget.LinearLayout;
import android.widget.TextView;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.annotation.RequiresApi;
import androidx.webkit.WebSettingsCompat;
import androidx.webkit.WebViewCompat;
import androidx.webkit.WebViewFeature;

import com.pichillilorenzo.flutter_inappwebview_android.InAppWebViewFlutterPlugin;
import com.pichillilorenzo.flutter_inappwebview_android.R;
import com.pichillilorenzo.flutter_inappwebview_android.Util;
import com.pichillilorenzo.flutter_inappwebview_android.content_blocker.ContentBlocker;
import com.pichillilorenzo.flutter_inappwebview_android.content_blocker.ContentBlockerAction;
import com.pichillilorenzo.flutter_inappwebview_android.content_blocker.ContentBlockerHandler;
import com.pichillilorenzo.flutter_inappwebview_android.content_blocker.ContentBlockerTrigger;
import com.pichillilorenzo.flutter_inappwebview_android.find_interaction.FindInteractionController;
import com.pichillilorenzo.flutter_inappwebview_android.in_app_browser.InAppBrowserDelegate;
import com.pichillilorenzo.flutter_inappwebview_android.plugin_scripts_js.InterceptAjaxRequestJS;
import com.pichillilorenzo.flutter_inappwebview_android.plugin_scripts_js.InterceptFetchRequestJS;
import com.pichillilorenzo.flutter_inappwebview_android.plugin_scripts_js.JavaScriptBridgeJS;
import com.pichillilorenzo.flutter_inappwebview_android.plugin_scripts_js.OnLoadResourceJS;
import com.pichillilorenzo.flutter_inappwebview_android.plugin_scripts_js.OnWindowBlurEventJS;
import com.pichillilorenzo.flutter_inappwebview_android.plugin_scripts_js.OnWindowFocusEventJS;
import com.pichillilorenzo.flutter_inappwebview_android.plugin_scripts_js.PluginScriptsUtil;
import com.pichillilorenzo.flutter_inappwebview_android.plugin_scripts_js.PrintJS;
import com.pichillilorenzo.flutter_inappwebview_android.plugin_scripts_js.PromisePolyfillJS;
import com.pichillilorenzo.flutter_inappwebview_android.print_job.PrintJobController;
import com.pichillilorenzo.flutter_inappwebview_android.print_job.PrintJobSettings;
import com.pichillilorenzo.flutter_inappwebview_android.pull_to_refresh.PullToRefreshLayout;
import com.pichillilorenzo.flutter_inappwebview_android.types.ContentWorld;
import com.pichillilorenzo.flutter_inappwebview_android.types.DownloadStartRequest;
import com.pichillilorenzo.flutter_inappwebview_android.types.PluginScript;
import com.pichillilorenzo.flutter_inappwebview_android.types.PreferredContentModeOptionType;
import com.pichillilorenzo.flutter_inappwebview_android.types.URLRequest;
import com.pichillilorenzo.flutter_inappwebview_android.types.UserContentController;
import com.pichillilorenzo.flutter_inappwebview_android.types.UserScript;
import com.pichillilorenzo.flutter_inappwebview_android.types.WebViewAssetLoaderExt;
import com.pichillilorenzo.flutter_inappwebview_android.webview.ContextMenuSettings;
import com.pichillilorenzo.flutter_inappwebview_android.webview.InAppWebViewInterface;
import com.pichillilorenzo.flutter_inappwebview_android.webview.JavaScriptBridgeInterface;
import com.pichillilorenzo.flutter_inappwebview_android.webview.WebViewChannelDelegate;
import com.pichillilorenzo.flutter_inappwebview_android.webview.web_message.WebMessageChannel;
import com.pichillilorenzo.flutter_inappwebview_android.webview.web_message.WebMessageListener;

import org.json.JSONObject;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.regex.Pattern;

import io.flutter.plugin.common.MethodChannel;

final public class InAppWebView extends InputAwareWebView implements InAppWebViewInterface {
  protected static final String LOG_TAG = "InAppWebView";
  public static final String METHOD_CHANNEL_NAME_PREFIX = "com.pichillilorenzo/flutter_inappwebview_";

  @Nullable
  public InAppWebViewFlutterPlugin plugin;
  @Nullable
  public InAppBrowserDelegate inAppBrowserDelegate;
  public Object id;
  @Nullable
  public Integer windowId;
  @Nullable
  public InAppWebViewClient inAppWebViewClient;
  @Nullable
  public InAppWebViewClientCompat inAppWebViewClientCompat;
  @Nullable
  public InAppWebViewChromeClient inAppWebViewChromeClient;
  @Nullable
  public InAppWebViewRenderProcessClient inAppWebViewRenderProcessClient;
  @Nullable
  public WebViewChannelDelegate channelDelegate;
  @Nullable
  public JavaScriptBridgeInterface javaScriptBridgeInterface;
  public InAppWebViewSettings customSettings = new InAppWebViewSettings();
  public boolean isLoading = false;
  private boolean inFullscreen = false;
  public float zoomScale = 1.0f;
  public ContentBlockerHandler contentBlockerHandler = new ContentBlockerHandler();
  public Pattern regexToCancelSubFramesLoadingCompiled;
  @Nullable
  public GestureDetector gestureDetector = null;
  @Nullable
  public LinearLayout floatingContextMenu = null;
  @Nullable
  public Map<String, Object> contextMenu = null;
  public Handler mainLooperHandler = new Handler(getWebViewLooper());
  static Handler mHandler = new Handler();

  public Runnable checkScrollStoppedTask;
  public int initialPositionScrollStoppedTask;
  public int newCheckScrollStoppedTask = 100; // ms

  public Runnable checkContextMenuShouldBeClosedTask;
  public int newCheckContextMenuShouldBeClosedTaskTask = 100; // ms

  public UserContentController userContentController = new UserContentController(this);

  public Map<String, ValueCallback<String>> callAsyncJavaScriptCallbacks = new HashMap<>();
  public Map<String, ValueCallback<String>> evaluateJavaScriptContentWorldCallbacks = new HashMap<>();

  public Map<String, WebMessageChannel> webMessageChannels = new HashMap<>();
  public List<WebMessageListener> webMessageListeners = new ArrayList<>();

  private List<UserScript> initialUserOnlyScripts = new ArrayList<>();

  @Nullable
  public FindInteractionController findInteractionController;

  @Nullable
  public WebViewAssetLoaderExt webViewAssetLoaderExt;

  @Nullable
  private PluginScript interceptOnlyAsyncAjaxRequestsPluginScript;

  public InAppWebView(Context context) {
    super(context);
  }

  public InAppWebView(Context context, AttributeSet attrs) {
    super(context, attrs);
  }

  public InAppWebView(Context context, AttributeSet attrs, int defaultStyle) {
    super(context, attrs, defaultStyle);
  }

  public InAppWebView(Context context, @NonNull InAppWebViewFlutterPlugin plugin,
                      @NonNull Object id, @Nullable Integer windowId, InAppWebViewSettings customSettings,
                      @Nullable Map<String, Object> contextMenu, View containerView,
                      List<UserScript> userScripts) {
    super(context, containerView, customSettings.useHybridComposition);
    if (!customSettings.gopeedProfileId.isEmpty()) {
      WebViewCompat.setProfile(this, customSettings.gopeedProfileId);
    }
    this.plugin = plugin;
    this.id = id;
    final MethodChannel channel = new MethodChannel(plugin.messenger, METHOD_CHANNEL_NAME_PREFIX + id);
    this.channelDelegate = new WebViewChannelDelegate(this, channel);
    this.windowId = windowId;
    this.customSettings = customSettings;
    this.contextMenu = contextMenu;
    this.initialUserOnlyScripts = userScripts;
    if (plugin != null && plugin.activity != null) {
      plugin.activity.registerForContextMenu(this);
    }
  }

  public WebViewClient createWebViewClient(InAppBrowserDelegate inAppBrowserDelegate) {
    // bug https://bugs.chromium.org/p/chromium/issues/detail?id=925887
    PackageInfo packageInfo = WebViewCompat.getCurrentWebViewPackage(getContext());
    if (packageInfo == null) {
      Log.d(LOG_TAG, "Using InAppWebViewClient implementation");
      return new InAppWebViewClient(inAppBrowserDelegate);
    }

    boolean isChromiumWebView = "com.android.webview".equals(packageInfo.packageName) ||
                                "com.google.android.webview".equals(packageInfo.packageName) ||
                                "com.android.chrome".equals(packageInfo.packageName);
    boolean isChromiumWebViewBugFixed = false;
    if (isChromiumWebView) {
      String versionName = packageInfo.versionName != null ? packageInfo.versionName : "";
      try {
        int majorVersion = versionName.contains(".") ?
                Integer.parseInt(versionName.split("\\.")[0]) : 0;
        isChromiumWebViewBugFixed = majorVersion >= 73;
      } catch (NumberFormatException ignored) {}
    }

    if (isChromiumWebViewBugFixed || !isChromiumWebView) {
      Log.d(LOG_TAG, "Using InAppWebViewClientCompat implementation");
      return new InAppWebViewClientCompat(inAppBrowserDelegate);
    } else {
      Log.d(LOG_TAG, "Using InAppWebViewClient implementation");
      return new InAppWebViewClient(inAppBrowserDelegate);
    }
  }

  @SuppressLint("RestrictedApi")
  private CookieManager gopeedCookieManager() {
    if (!customSettings.gopeedProfileId.isEmpty()) {
      return WebViewCompat.getProfile(this).getCookieManager();
    }
    return CookieManager.getInstance();
  }

  public void prepare() {
    if (plugin != null) {
      webViewAssetLoaderExt = WebViewAssetLoaderExt.fromMap(customSettings.webViewAssetLoader, plugin, getContext());
    }

    javaScriptBridgeInterface = new JavaScriptBridgeInterface(this);
    addJavascriptInterface(javaScriptBridgeInterface, JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME);

    inAppWebViewChromeClient = new InAppWebViewChromeClient(plugin, this, inAppBrowserDelegate);
    setWebChromeClient(inAppWebViewChromeClient);

    WebViewClient webViewClient = createWebViewClient(inAppBrowserDelegate);
    if (webViewClient instanceof InAppWebViewClientCompat) {
      inAppWebViewClientCompat = (InAppWebViewClientCompat) webViewClient;
      setWebViewClient(inAppWebViewClientCompat);
    } else if (webViewClient instanceof InAppWebViewClient) {
      inAppWebViewClient = (InAppWebViewClient) webViewClient;
      setWebViewClient(inAppWebViewClient);
    }

    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q && WebViewFeature.isFeatureSupported(WebViewFeature.WEB_VIEW_RENDERER_CLIENT_BASIC_USAGE)) {
      inAppWebViewRenderProcessClient = new InAppWebViewRenderProcessClient();
      WebViewCompat.setWebViewRenderProcessClient(this, inAppWebViewRenderProcessClient);
    }

    if (windowId == null || !WebViewFeature.isFeatureSupported(WebViewFeature.DOCUMENT_START_SCRIPT)) {
      // for some reason, if a WebView is created using a window id,
      // the initial plugin and user scripts injected
      // with WebViewCompat.addDocumentStartJavaScript will not be added!
      // https://github.com/pichillilorenzo/flutter_inappwebview/issues/1455
      prepareAndAddUserScripts();
    }

    if (customSettings.useOnDownloadStart)
      setDownloadListener(new DownloadStartListener());

    WebSettings settings = getSettings();

    settings.setJavaScriptEnabled(customSettings.javaScriptEnabled);
    settings.setJavaScriptCanOpenWindowsAutomatically(customSettings.javaScriptCanOpenWindowsAutomatically);
    settings.setBuiltInZoomControls(customSettings.builtInZoomControls);
    settings.setDisplayZoomControls(customSettings.displayZoomControls);
    settings.setSupportMultipleWindows(customSettings.supportMultipleWindows);

    if (WebViewFeature.isFeatureSupported(WebViewFeature.SAFE_BROWSING_ENABLE))
      WebSettingsCompat.setSafeBrowsingEnabled(settings, customSettings.safeBrowsingEnabled);
    else if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O)
      settings.setSafeBrowsingEnabled(customSettings.safeBrowsingEnabled);

    settings.setMediaPlaybackRequiresUserGesture(customSettings.mediaPlaybackRequiresUserGesture);

    settings.setDatabaseEnabled(customSettings.databaseEnabled);
    settings.setDomStorageEnabled(customSettings.domStorageEnabled);

    if (customSettings.userAgent != null && !customSettings.userAgent.isEmpty())
      settings.setUserAgentString(customSettings.userAgent);
    else if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.JELLY_BEAN_MR1)
      settings.setUserAgentString(WebSettings.getDefaultUserAgent(getContext()));

    if (customSettings.applicationNameForUserAgent != null && !customSettings.applicationNameForUserAgent.isEmpty()) {
      if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.JELLY_BEAN_MR1) {
        String userAgent = (customSettings.userAgent != null && !customSettings.userAgent.isEmpty()) ? customSettings.userAgent : WebSettings.getDefaultUserAgent(getContext());
        String userAgentWithApplicationName = userAgent + " " + customSettings.applicationNameForUserAgent;
        settings.setUserAgentString(userAgentWithApplicationName);
      }
    }

    if (customSettings.clearCache)
      clearAllCache();
    else if (customSettings.clearSessionCache)
      gopeedCookieManager().removeSessionCookie();

    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP)
      gopeedCookieManager().setAcceptThirdPartyCookies(this, customSettings.thirdPartyCookiesEnabled);

    settings.setLoadWithOverviewMode(customSettings.loadWithOverviewMode);
    settings.setUseWideViewPort(customSettings.useWideViewPort);
    settings.setSupportZoom(customSettings.supportZoom);
    settings.setTextZoom(customSettings.textZoom);

    setVerticalScrollBarEnabled(!customSettings.disableVerticalScroll && customSettings.verticalScrollBarEnabled);
    setHorizontalScrollBarEnabled(!customSettings.disableHorizontalScroll && customSettings.horizontalScrollBarEnabled);

    if (customSettings.transparentBackground)
      setBackgroundColor(Color.TRANSPARENT);

    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP && customSettings.mixedContentMode != null)
      settings.setMixedContentMode(customSettings.mixedContentMode);

    settings.setAllowContentAccess(customSettings.allowContentAccess);
    settings.setAllowFileAccess(customSettings.allowFileAccess);
    settings.setAllowFileAccessFromFileURLs(customSettings.allowFileAccessFromFileURLs);
    settings.setAllowUniversalAccessFromFileURLs(customSettings.allowUniversalAccessFromFileURLs);
    setCacheEnabled(customSettings.cacheEnabled);
    if (customSettings.appCachePath != null && !customSettings.appCachePath.isEmpty() && customSettings.cacheEnabled) {
      // removed from Android API 33+ (https://developer.android.com/sdk/api_diff/33/changes)
      // settings.setAppCachePath(customSettings.appCachePath);
      Util.invokeMethodIfExists(settings, "setAppCachePath", customSettings.appCachePath);
    }
    settings.setBlockNetworkImage(customSettings.blockNetworkImage);
    settings.setBlockNetworkLoads(customSettings.blockNetworkLoads);
    if (customSettings.cacheMode != null)
      settings.setCacheMode(customSettings.cacheMode);
    settings.setCursiveFontFamily(customSettings.cursiveFontFamily);
    settings.setDefaultFixedFontSize(customSettings.defaultFixedFontSize);
    settings.setDefaultFontSize(customSettings.defaultFontSize);
    settings.setDefaultTextEncodingName(customSettings.defaultTextEncodingName);
    if (customSettings.disabledActionModeMenuItems != null) {
      if (WebViewFeature.isFeatureSupported(WebViewFeature.DISABLED_ACTION_MODE_MENU_ITEMS))
        WebSettingsCompat.setDisabledActionModeMenuItems(settings, customSettings.disabledActionModeMenuItems);
      else if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.N)
        settings.setDisabledActionModeMenuItems(customSettings.disabledActionModeMenuItems);
    }
    settings.setFantasyFontFamily(customSettings.fantasyFontFamily);
    settings.setFixedFontFamily(customSettings.fixedFontFamily);
    if (customSettings.forceDark != null) {
      if (WebViewFeature.isFeatureSupported(WebViewFeature.FORCE_DARK))
        WebSettingsCompat.setForceDark(settings, customSettings.forceDark);
      else if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q)
        settings.setForceDark(customSettings.forceDark);
    }
    if (customSettings.forceDarkStrategy != null && WebViewFeature.isFeatureSupported(WebViewFeature.FORCE_DARK_STRATEGY)) {
      WebSettingsCompat.setForceDarkStrategy(settings, customSettings.forceDarkStrategy);
    }
    settings.setGeolocationEnabled(customSettings.geolocationEnabled);
    if (customSettings.layoutAlgorithm != null) {
      if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.KITKAT && customSettings.layoutAlgorithm.equals(WebSettings.LayoutAlgorithm.TEXT_AUTOSIZING)) {
        settings.setLayoutAlgorithm(customSettings.layoutAlgorithm);
      } else {
        settings.setLayoutAlgorithm(customSettings.layoutAlgorithm);
      }
    }
    settings.setLoadsImagesAutomatically(customSettings.loadsImagesAutomatically);
    settings.setMinimumFontSize(customSettings.minimumFontSize);
    settings.setMinimumLogicalFontSize(customSettings.minimumLogicalFontSize);
    setInitialScale(customSettings.initialScale);
    settings.setNeedInitialFocus(customSettings.needInitialFocus);
    if (WebViewFeature.isFeatureSupported(WebViewFeature.OFF_SCREEN_PRERASTER))
      WebSettingsCompat.setOffscreenPreRaster(settings, customSettings.offscreenPreRaster);
    else if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M)
      settings.setOffscreenPreRaster(customSettings.offscreenPreRaster);
    settings.setSansSerifFontFamily(customSettings.sansSerifFontFamily);
    settings.setSerifFontFamily(customSettings.serifFontFamily);
    settings.setStandardFontFamily(customSettings.standardFontFamily);
    if (customSettings.preferredContentMode != null &&
            customSettings.preferredContentMode == PreferredContentModeOptionType.DESKTOP.toValue()) {
      setDesktopMode(true);
    }
    settings.setSaveFormData(customSettings.saveFormData);
    if (customSettings.incognito)
      setIncognito(true);
    if (customSettings.useHybridComposition) {
      if (customSettings.hardwareAcceleration)
        setLayerType(View.LAYER_TYPE_HARDWARE, null);
      else
        setLayerType(View.LAYER_TYPE_NONE, null);
    }
    if (customSettings.regexToCancelSubFramesLoading != null) {
      regexToCancelSubFramesLoadingCompiled = Pattern.compile(customSettings.regexToCancelSubFramesLoading);
    }
    setScrollBarStyle(customSettings.scrollBarStyle);
    if (customSettings.scrollBarDefaultDelayBeforeFade != null) {
      setScrollBarDefaultDelayBeforeFade(customSettings.scrollBarDefaultDelayBeforeFade);
    } else {
      customSettings.scrollBarDefaultDelayBeforeFade = getScrollBarDefaultDelayBeforeFade();
    }
    setScrollbarFadingEnabled(customSettings.scrollbarFadingEnabled);
    if (customSettings.scrollBarFadeDuration != null) {
      setScrollBarFadeDuration(customSettings.scrollBarFadeDuration);
    } else {
      customSettings.scrollBarFadeDuration = getScrollBarFadeDuration();
    }
    setVerticalScrollbarPosition(customSettings.verticalScrollbarPosition);

    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
      if (customSettings.verticalScrollbarThumbColor != null)
        setVerticalScrollbarThumbDrawable(new ColorDrawable(Color.parseColor(customSettings.verticalScrollbarThumbColor)));
      if (customSettings.verticalScrollbarTrackColor != null)
        setVerticalScrollbarTrackDrawable(new ColorDrawable(Color.parseColor(customSettings.verticalScrollbarTrackColor)));
      if (customSettings.horizontalScrollbarThumbColor != null)
        setHorizontalScrollbarThumbDrawable(new ColorDrawable(Color.parseColor(customSettings.horizontalScrollbarThumbColor)));
      if (customSettings.horizontalScrollbarTrackColor != null)
        setHorizontalScrollbarTrackDrawable(new ColorDrawable(Color.parseColor(customSettings.horizontalScrollbarTrackColor)));
    }

    setOverScrollMode(customSettings.overScrollMode);
    if (customSettings.networkAvailable != null) {
      setNetworkAvailable(customSettings.networkAvailable);
    }
    if (customSettings.rendererPriorityPolicy != null && !customSettings.rendererPriorityPolicy.isEmpty() && Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
      setRendererPriorityPolicy(
              (int) customSettings.rendererPriorityPolicy.get("rendererRequestedPriority"),
              (boolean) customSettings.rendererPriorityPolicy.get("waivedWhenNotVisible"));
    }

    if (WebViewFeature.isFeatureSupported(WebViewFeature.ALGORITHMIC_DARKENING) && Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
      WebSettingsCompat.setAlgorithmicDarkeningAllowed(settings, customSettings.algorithmicDarkeningAllowed);
    }
    if (WebViewFeature.isFeatureSupported(WebViewFeature.ENTERPRISE_AUTHENTICATION_APP_LINK_POLICY)) {
      WebSettingsCompat.setEnterpriseAuthenticationAppLinkPolicyEnabled(settings, customSettings.enterpriseAuthenticationAppLinkPolicyEnabled);
    }
    if (customSettings.requestedWithHeaderOriginAllowList != null &&
            WebViewFeature.isFeatureSupported(WebViewFeature.REQUESTED_WITH_HEADER_ALLOW_LIST)) {
      WebSettingsCompat.setRequestedWithHeaderOriginAllowList(settings, customSettings.requestedWithHeaderOriginAllowList);
    }

    contentBlockerHandler.getRuleList().clear();
    for (Map<String, Map<String, Object>> contentBlocker : customSettings.contentBlockers) {
      // compile ContentBlockerTrigger urlFilter
      ContentBlockerTrigger trigger = ContentBlockerTrigger.fromMap(contentBlocker.get("trigger"));
      ContentBlockerAction action = ContentBlockerAction.fromMap(contentBlocker.get("action"));
      contentBlockerHandler.getRuleList().add(new ContentBlocker(trigger, action));
    }

    setFindListener(new FindListener() {
      @Override
      public void onFindResultReceived(int activeMatchOrdinal, int numberOfMatches, boolean isDoneCounting) {
        if (findInteractionController != null && findInteractionController.channelDelegate != null)
          findInteractionController.channelDelegate.onFindResultReceived(activeMatchOrdinal, numberOfMatches, isDoneCounting);
        if (channelDelegate != null)
          channelDelegate.onFindResultReceived(activeMatchOrdinal, numberOfMatches, isDoneCounting);
      }
    });

    gestureDetector = new GestureDetector(this.getContext(), new GestureDetector.SimpleOnGestureListener() {
      @Override
      public boolean onSingleTapUp(MotionEvent ev) {
        if (floatingContextMenu != null) {
          hideContextMenu();
        }
        return super.onSingleTapUp(ev);
      }
    });

    checkScrollStoppedTask = new Runnable() {
      @Override
      public void run() {
        int newPosition = getScrollY();
        if (initialPositionScrollStoppedTask - newPosition == 0) {
          // has stopped
          onScrollStopped();
        } else {
          initialPositionScrollStoppedTask = getScrollY();
          mainLooperHandler.postDelayed(checkScrollStoppedTask, newCheckScrollStoppedTask);
        }
      }
    };

    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.KITKAT && !customSettings.useHybridComposition) {
      checkContextMenuShouldBeClosedTask = new Runnable() {
        @Override
        public void run() {
          if (floatingContextMenu != null) {
            evaluateJavascript(PluginScriptsUtil.CHECK_CONTEXT_MENU_SHOULD_BE_HIDDEN_JS_SOURCE, new ValueCallback<String>() {
              @Override
              public void onReceiveValue(String value) {
                if (value == null || value.equals("true")) {
                  if (floatingContextMenu != null) {
                    hideContextMenu();
                  }
                } else {
                  mainLooperHandler.postDelayed(checkContextMenuShouldBeClosedTask, newCheckContextMenuShouldBeClosedTaskTask);
                }
              }
            });
          }
        }
      };
    }

    setOnTouchListener(new OnTouchListener() {
      float m_downX;
      float m_downY;

      @Override
      public boolean onTouch(View v, MotionEvent event) {
        gestureDetector.onTouchEvent(event);

        if (event.getAction() == MotionEvent.ACTION_UP) {
          checkScrollStoppedTask.run();
        }

        if (customSettings.disableHorizontalScroll && customSettings.disableVerticalScroll) {
          return (event.getAction() == MotionEvent.ACTION_MOVE);
        } else if (customSettings.disableHorizontalScroll || customSettings.disableVerticalScroll) {
          switch (event.getAction()) {
            case MotionEvent.ACTION_DOWN: {
              // save the x
              m_downX = event.getX();
              // save the y
              m_downY = event.getY();
              break;
            }
            case MotionEvent.ACTION_MOVE:
            case MotionEvent.ACTION_CANCEL:
            case MotionEvent.ACTION_UP: {
              if (customSettings.disableHorizontalScroll) {
                // set x so that it doesn't move
                event.setLocation(m_downX, event.getY());
              } else {
                // set y so that it doesn't move
                event.setLocation(event.getX(), m_downY);
              }
              break;
            }
          }
        }
        return false;
      }
    });

    setOnLongClickListener(new OnLongClickListener() {
      @Override
      public boolean onLongClick(View v) {
        com.pichillilorenzo.flutter_inappwebview_android.types.HitTestResult hitTestResult =
                com.pichillilorenzo.flutter_inappwebview_android.types.HitTestResult.fromWebViewHitTestResult(getHitTestResult());
        if (channelDelegate != null) channelDelegate.onLongPressHitTestResult(hitTestResult);
        return false;
      }
    });
  }

  public void prepareAndAddUserScripts() {
    userContentController.addPluginScript(PromisePolyfillJS.PROMISE_POLYFILL_JS_PLUGIN_SCRIPT);
    userContentController.addPluginScript(JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_JS_PLUGIN_SCRIPT);
    userContentController.addPluginScript(PrintJS.PRINT_JS_PLUGIN_SCRIPT);
    userContentController.addPluginScript(OnWindowBlurEventJS.ON_WINDOW_BLUR_EVENT_JS_PLUGIN_SCRIPT);
    userContentController.addPluginScript(OnWindowFocusEventJS.ON_WINDOW_FOCUS_EVENT_JS_PLUGIN_SCRIPT);
    interceptOnlyAsyncAjaxRequestsPluginScript = InterceptAjaxRequestJS.createInterceptOnlyAsyncAjaxRequestsPluginScript(customSettings.interceptOnlyAsyncAjaxRequests);
    if (customSettings.useShouldInterceptAjaxRequest) {
      userContentController.addPluginScript(interceptOnlyAsyncAjaxRequestsPluginScript);
      userContentController.addPluginScript(InterceptAjaxRequestJS.INTERCEPT_AJAX_REQUEST_JS_PLUGIN_SCRIPT);
    }
    if (customSettings.useShouldInterceptFetchRequest) {
      userContentController.addPluginScript(InterceptFetchRequestJS.INTERCEPT_FETCH_REQUEST_JS_PLUGIN_SCRIPT);
    }
    if (customSettings.useOnLoadResource) {
      userContentController.addPluginScript(OnLoadResourceJS.ON_LOAD_RESOURCE_JS_PLUGIN_SCRIPT);
    }
    if (!customSettings.useHybridComposition) {
      userContentController.addPluginScript(PluginScriptsUtil.CHECK_GLOBAL_KEY_DOWN_EVENT_TO_HIDE_CONTEXT_MENU_JS_PLUGIN_SCRIPT);
    }
    this.userContentController.addUserOnlyScripts(this.initialUserOnlyScripts);
  }

  public void setIncognito(boolean enabled) {
    WebSettings settings = getSettings();
    if (enabled) {
      if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
        gopeedCookieManager().removeAllCookies(null);
      } else {
        gopeedCookieManager().removeAllCookie();
      }

      // Disable caching
      settings.setCacheMode(WebSettings.LOAD_NO_CACHE);

      // removed from Android API 33+ (https://developer.android.com/sdk/api_diff/33/changes)
      // settings.setAppCacheEnabled(false);
      Util.invokeMethodIfExists(settings, "setAppCacheEnabled", false);

      clearHistory();
      clearCache(true);

      // No form data or autofill enabled
      clearFormData();
      settings.setSavePassword(false);
      settings.setSaveFormData(false);
    } else {
      settings.setCacheMode(WebSettings.LOAD_DEFAULT);

      // removed from Android API 33+ (https://developer.android.com/sdk/api_diff/33/changes)
      // settings.setAppCacheEnabled(true);
      Util.invokeMethodIfExists(settings, "setAppCacheEnabled", true);

      settings.setSavePassword(true);
      settings.setSaveFormData(true);
    }
  }

  public void setCacheEnabled(boolean enabled) {
    WebSettings settings = getSettings();
    if (enabled) {
      Context ctx = getContext();
      if (ctx != null) {
        // removed from Android API 33+ (https://developer.android.com/sdk/api_diff/33/changes)
        // settings.setAppCachePath(ctx.getCacheDir().getAbsolutePath());
        Util.invokeMethodIfExists(settings, "setAppCachePath", ctx.getCacheDir().getAbsolutePath());

        settings.setCacheMode(WebSettings.LOAD_DEFAULT);

        // removed from Android API 33+ (https://developer.android.com/sdk/api_diff/33/changes)
        // settings.setAppCacheEnabled(true);
        Util.invokeMethodIfExists(settings, "setAppCacheEnabled", true);
      }
    } else {
      settings.setCacheMode(WebSettings.LOAD_NO_CACHE);

      // removed from Android API 33+ (https://developer.android.com/sdk/api_diff/33/changes)
      // settings.setAppCacheEnabled(false);
      Util.invokeMethodIfExists(settings, "setAppCacheEnabled", false);
    }
  }

  public void loadUrl(URLRequest urlRequest) {
    String url = urlRequest.getUrl();
    String method = urlRequest.getMethod();
    if (method != null && method.equals("POST")) {
      byte[] postData = urlRequest.getBody();
      postUrl(url, postData);
      return;
    }
    Map<String, String> headers = urlRequest.getHeaders();
    if (headers != null) {
      loadUrl(url, headers);
      return;
    }
    loadUrl(url);
  }

  public void loadFile(String assetFilePath) throws IOException {
    if (plugin == null) {
      return;
    }

    loadUrl(Util.getUrlAsset(plugin, assetFilePath));
  }

  public boolean isLoading() {
    return isLoading;
  }

  /**
   * @deprecated
   */
  @Deprecated
  private void clearCookies() {
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
      gopeedCookieManager().removeAllCookies(new ValueCallback<Boolean>() {
        @Override
        public void onReceiveValue(Boolean aBoolean) {

        }
      });
    } else {
      gopeedCookieManager().removeAllCookie();
    }
  }

  /**
   * @deprecated
   */
  @Deprecated
  public void clearAllCache() {
    clearCache(true);
    clearCookies();
    clearFormData();
    WebStorage.getInstance().deleteAllData();
  }

  public void takeScreenshot(final @Nullable Map<String, Object> screenshotConfiguration, final MethodChannel.Result result) {
    final float pixelDensity = Util.getPixelDensity(getContext());

    mainLooperHandler.post(new Runnable() {
      @Override
      public void run() {
        try {
          Bitmap screenshotBitmap = Bitmap.createBitmap(getMeasuredWidth(), getMeasuredHeight(), Bitmap.Config.ARGB_8888);
          Canvas c = new Canvas(screenshotBitmap);
          c.translate(-getScrollX(), -getScrollY());
          draw(c);

          ByteArrayOutputStream byteArrayOutputStream = new ByteArrayOutputStream();
          Bitmap.CompressFormat compressFormat = Bitmap.CompressFormat.PNG;
          int quality = 100;

          if (screenshotConfiguration != null) {
            Map<String, Double> rect = (Map<String, Double>) screenshotConfiguration.get("rect");
            if (rect != null) {
              int rectX = (int) Math.floor(rect.get("x") * pixelDensity + 0.5);
              int rectY = (int) Math.floor(rect.get("y") * pixelDensity + 0.5);
              int rectWidth = Math.min(screenshotBitmap.getWidth(), (int) Math.floor(rect.get("width") * pixelDensity + 0.5));
              int rectHeight = Math.min(screenshotBitmap.getHeight(), (int) Math.floor(rect.get("height") * pixelDensity + 0.5));
              screenshotBitmap = Bitmap.createBitmap(
                      screenshotBitmap,
                      rectX,
                      rectY,
                      rectWidth,
                      rectHeight);
            }

            Double snapshotWidth = (Double) screenshotConfiguration.get("snapshotWidth");
            if (snapshotWidth != null) {
              int dstWidth = (int) Math.floor(snapshotWidth * pixelDensity + 0.5);
              float ratioBitmap = (float) screenshotBitmap.getWidth() / (float) screenshotBitmap.getHeight();
              int dstHeight = (int) ((float) dstWidth / ratioBitmap);
              screenshotBitmap = Bitmap.createScaledBitmap(screenshotBitmap, dstWidth, dstHeight, true);
            }

            try {
              compressFormat = Bitmap.CompressFormat.valueOf((String) screenshotConfiguration.get("compressFormat"));
            } catch (IllegalArgumentException e) {
              Log.e(LOG_TAG, "", e);
            }

            quality = (Integer) screenshotConfiguration.get("quality");
          }

          screenshotBitmap.compress(
                  compressFormat,
                  quality,
                  byteArrayOutputStream);

          try {
            byteArrayOutputStream.close();
          } catch (IOException e) {
            Log.e(LOG_TAG, "", e);
          }
          screenshotBitmap.recycle();
          result.success(byteArrayOutputStream.toByteArray());

        } catch (IllegalArgumentException e) {
          Log.e(LOG_TAG, "", e);
          result.success(null);
        }
      }
    });
  }

  @SuppressLint("RestrictedApi")
  public void setSettings(InAppWebViewSettings newCustomSettings, HashMap<String, Object> newSettingsMap) {

    WebSettings settings = getSettings();

    if (newSettingsMap.get("javaScriptEnabled") != null && customSettings.javaScriptEnabled != newCustomSettings.javaScriptEnabled)
      settings.setJavaScriptEnabled(newCustomSettings.javaScriptEnabled);

    if (newSettingsMap.get("useShouldInterceptAjaxRequest") != null && customSettings.useShouldInterceptAjaxRequest != newCustomSettings.useShouldInterceptAjaxRequest) {
      enablePluginScriptAtRuntime(
              InterceptAjaxRequestJS.FLAG_VARIABLE_FOR_SHOULD_INTERCEPT_AJAX_REQUEST_JS_SOURCE,
              newCustomSettings.useShouldInterceptAjaxRequest,
              InterceptAjaxRequestJS.INTERCEPT_AJAX_REQUEST_JS_PLUGIN_SCRIPT
      );
    }

    if (newSettingsMap.get("interceptOnlyAsyncAjaxRequests") != null && customSettings.interceptOnlyAsyncAjaxRequests != newCustomSettings.interceptOnlyAsyncAjaxRequests) {
      enablePluginScriptAtRuntime(
              InterceptAjaxRequestJS.FLAG_VARIABLE_FOR_INTERCEPT_ONLY_ASYNC_AJAX_REQUESTS_JS_SOURCE,
              newCustomSettings.interceptOnlyAsyncAjaxRequests,
              interceptOnlyAsyncAjaxRequestsPluginScript
      );
    }

    if (newSettingsMap.get("useShouldInterceptFetchRequest") != null && customSettings.useShouldInterceptFetchRequest != newCustomSettings.useShouldInterceptFetchRequest) {
      enablePluginScriptAtRuntime(
              InterceptFetchRequestJS.FLAG_VARIABLE_FOR_SHOULD_INTERCEPT_FETCH_REQUEST_JS_SOURCE,
              newCustomSettings.useShouldInterceptFetchRequest,
              InterceptFetchRequestJS.INTERCEPT_FETCH_REQUEST_JS_PLUGIN_SCRIPT
      );
    }

    if (newSettingsMap.get("useOnLoadResource") != null && customSettings.useOnLoadResource != newCustomSettings.useOnLoadResource) {
      enablePluginScriptAtRuntime(
              OnLoadResourceJS.FLAG_VARIABLE_FOR_ON_LOAD_RESOURCE_JS_SOURCE,
              newCustomSettings.useOnLoadResource,
              OnLoadResourceJS.ON_LOAD_RESOURCE_JS_PLUGIN_SCRIPT
      );
    }

    if (newSettingsMap.get("javaScriptCanOpenWindowsAutomatically") != null && customSettings.javaScriptCanOpenWindowsAutomatically != newCustomSettings.javaScriptCanOpenWindowsAutomatically)
      settings.setJavaScriptCanOpenWindowsAutomatically(newCustomSettings.javaScriptCanOpenWindowsAutomatically);

    if (newSettingsMap.get("builtInZoomControls") != null && customSettings.builtInZoomControls != newCustomSettings.builtInZoomControls)
      settings.setBuiltInZoomControls(newCustomSettings.builtInZoomControls);

    if (newSettingsMap.get("displayZoomControls") != null && customSettings.displayZoomControls != newCustomSettings.displayZoomControls)
      settings.setDisplayZoomControls(newCustomSettings.displayZoomControls);

    if (newSettingsMap.get("safeBrowsingEnabled") != null && customSettings.safeBrowsingEnabled != newCustomSettings.safeBrowsingEnabled) {
      if (WebViewFeature.isFeatureSupported(WebViewFeature.SAFE_BROWSING_ENABLE))
        WebSettingsCompat.setSafeBrowsingEnabled(settings, newCustomSettings.safeBrowsingEnabled);
      else if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O)
        settings.setSafeBrowsingEnabled(newCustomSettings.safeBrowsingEnabled);
    }

    if (newSettingsMap.get("mediaPlaybackRequiresUserGesture") != null && customSettings.mediaPlaybackRequiresUserGesture != newCustomSettings.mediaPlaybackRequiresUserGesture)
      settings.setMediaPlaybackRequiresUserGesture(newCustomSettings.mediaPlaybackRequiresUserGesture);

    if (newSettingsMap.get("databaseEnabled") != null && customSettings.databaseEnabled != newCustomSettings.databaseEnabled)
      settings.setDatabaseEnabled(newCustomSettings.databaseEnabled);

    if (newSettingsMap.get("domStorageEnabled") != null && customSettings.domStorageEnabled != newCustomSettings.domStorageEnabled)
      settings.setDomStorageEnabled(newCustomSettings.domStorageEnabled);

    if (newSettingsMap.get("userAgent") != null && !customSettings.userAgent.equals(newCustomSettings.userAgent) && !newCustomSettings.userAgent.isEmpty())
      settings.setUserAgentString(newCustomSettings.userAgent);

    if (newSettingsMap.get("applicationNameForUserAgent") != null && !customSettings.applicationNameForUserAgent.equals(newCustomSettings.applicationNameForUserAgent) && !newCustomSettings.applicationNameForUserAgent.isEmpty()) {
      if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.JELLY_BEAN_MR1) {
        String userAgent = (newCustomSettings.userAgent != null && !newCustomSettings.userAgent.isEmpty()) ? newCustomSettings.userAgent : WebSettings.getDefaultUserAgent(getContext());
        String userAgentWithApplicationName = userAgent + " " + customSettings.applicationNameForUserAgent;
        settings.setUserAgentString(userAgentWithApplicationName);
      }
    }

    if (newSettingsMap.get("clearCache") != null && newCustomSettings.clearCache)
      clearAllCache();
    else if (newSettingsMap.get("clearSessionCache") != null && newCustomSettings.clearSessionCache)
      gopeedCookieManager().removeSessionCookie();

    if (newSettingsMap.get("thirdPartyCookiesEnabled") != null && customSettings.thirdPartyCookiesEnabled != newCustomSettings.thirdPartyCookiesEnabled && Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP)
      gopeedCookieManager().setAcceptThirdPartyCookies(this, newCustomSettings.thirdPartyCookiesEnabled);

    if (newSettingsMap.get("useWideViewPort") != null && customSettings.useWideViewPort != newCustomSettings.useWideViewPort)
      settings.setUseWideViewPort(newCustomSettings.useWideViewPort);

    if (newSettingsMap.get("supportZoom") != null && customSettings.supportZoom != newCustomSettings.supportZoom)
      settings.setSupportZoom(newCustomSettings.supportZoom);

    if (newSettingsMap.get("textZoom") != null && !customSettings.textZoom.equals(newCustomSettings.textZoom))
      settings.setTextZoom(newCustomSettings.textZoom);

    if (newSettingsMap.get("verticalScrollBarEnabled") != null && customSettings.verticalScrollBarEnabled != newCustomSettings.verticalScrollBarEnabled)
      setVerticalScrollBarEnabled(newCustomSettings.verticalScrollBarEnabled);

    if (newSettingsMap.get("horizontalScrollBarEnabled") != null && customSettings.horizontalScrollBarEnabled != newCustomSettings.horizontalScrollBarEnabled)
      setHorizontalScrollBarEnabled(newCustomSettings.horizontalScrollBarEnabled);

    if (newSettingsMap.get("transparentBackground") != null && customSettings.transparentBackground != newCustomSettings.transparentBackground) {
      if (newCustomSettings.transparentBackground) {
        setBackgroundColor(Color.TRANSPARENT);
      } else {
        setBackgroundColor(Color.parseColor("#FFFFFF"));
      }
    }

    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP)
      if (newSettingsMap.get("mixedContentMode") != null && (customSettings.mixedContentMode == null || !customSettings.mixedContentMode.equals(newCustomSettings.mixedContentMode)))
        settings.setMixedContentMode(newCustomSettings.mixedContentMode);

    if (newSettingsMap.get("supportMultipleWindows") != null && customSettings.supportMultipleWindows != newCustomSettings.supportMultipleWindows)
      settings.setSupportMultipleWindows(newCustomSettings.supportMultipleWindows);

    if (newSettingsMap.get("useOnDownloadStart") != null && customSettings.useOnDownloadStart != newCustomSettings.useOnDownloadStart) {
      if (newCustomSettings.useOnDownloadStart) {
        setDownloadListener(new DownloadStartListener());
      } else {
        setDownloadListener(null);
      }
    }

    if (newSettingsMap.get("allowContentAccess") != null && customSettings.allowContentAccess != newCustomSettings.allowContentAccess)
      settings.setAllowContentAccess(newCustomSettings.allowContentAccess);

    if (newSettingsMap.get("allowFileAccess") != null && customSettings.allowFileAccess != newCustomSettings.allowFileAccess)
      settings.setAllowFileAccess(newCustomSettings.allowFileAccess);

    if (newSettingsMap.get("allowFileAccessFromFileURLs") != null && customSettings.allowFileAccessFromFileURLs != newCustomSettings.allowFileAccessFromFileURLs)
      settings.setAllowFileAccessFromFileURLs(newCustomSettings.allowFileAccessFromFileURLs);

    if (newSettingsMap.get("allowUniversalAccessFromFileURLs") != null && customSettings.allowUniversalAccessFromFileURLs != newCustomSettings.allowUniversalAccessFromFileURLs)
      settings.setAllowUniversalAccessFromFileURLs(newCustomSettings.allowUniversalAccessFromFileURLs);

    if (newSettingsMap.get("cacheEnabled") != null && customSettings.cacheEnabled != newCustomSettings.cacheEnabled)
      setCacheEnabled(newCustomSettings.cacheEnabled);

    if (newSettingsMap.get("appCachePath") != null && (customSettings.appCachePath == null || !customSettings.appCachePath.equals(newCustomSettings.appCachePath))) {
      // removed from Android API 33+ (https://developer.android.com/sdk/api_diff/33/changes)
      // settings.setAppCachePath(newCustomSettings.appCachePath);
      Util.invokeMethodIfExists(settings, "setAppCachePath", newCustomSettings.appCachePath);
    }

    if (newSettingsMap.get("blockNetworkImage") != null && customSettings.blockNetworkImage != newCustomSettings.blockNetworkImage)
      settings.setBlockNetworkImage(newCustomSettings.blockNetworkImage);

    if (newSettingsMap.get("blockNetworkLoads") != null && customSettings.blockNetworkLoads != newCustomSettings.blockNetworkLoads)
      settings.setBlockNetworkLoads(newCustomSettings.blockNetworkLoads);

    if (newSettingsMap.get("cacheMode") != null && !customSettings.cacheMode.equals(newCustomSettings.cacheMode))
      settings.setCacheMode(newCustomSettings.cacheMode);

    if (newSettingsMap.get("cursiveFontFamily") != null && !customSettings.cursiveFontFamily.equals(newCustomSettings.cursiveFontFamily))
      settings.setCursiveFontFamily(newCustomSettings.cursiveFontFamily);

    if (newSettingsMap.get("defaultFixedFontSize") != null && !customSettings.defaultFixedFontSize.equals(newCustomSettings.defaultFixedFontSize))
      settings.setDefaultFixedFontSize(newCustomSettings.defaultFixedFontSize);

    if (newSettingsMap.get("defaultFontSize") != null && !customSettings.defaultFontSize.equals(newCustomSettings.defaultFontSize))
      settings.setDefaultFontSize(newCustomSettings.defaultFontSize);

    if (newSettingsMap.get("defaultTextEncodingName") != null && !customSettings.defaultTextEncodingName.equals(newCustomSettings.defaultTextEncodingName))
      settings.setDefaultTextEncodingName(newCustomSettings.defaultTextEncodingName);

    if (newSettingsMap.get("disabledActionModeMenuItems") != null &&
            (customSettings.disabledActionModeMenuItems == null ||
            !customSettings.disabledActionModeMenuItems.equals(newCustomSettings.disabledActionModeMenuItems))) {
      if (WebViewFeature.isFeatureSupported(WebViewFeature.DISABLED_ACTION_MODE_MENU_ITEMS))
        WebSettingsCompat.setDisabledActionModeMenuItems(settings, newCustomSettings.disabledActionModeMenuItems);
      else if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.N)
        settings.setDisabledActionModeMenuItems(newCustomSettings.disabledActionModeMenuItems);
    }

    if (newSettingsMap.get("fantasyFontFamily") != null && !customSettings.fantasyFontFamily.equals(newCustomSettings.fantasyFontFamily))
      settings.setFantasyFontFamily(newCustomSettings.fantasyFontFamily);

    if (newSettingsMap.get("fixedFontFamily") != null && !customSettings.fixedFontFamily.equals(newCustomSettings.fixedFontFamily))
      settings.setFixedFontFamily(newCustomSettings.fixedFontFamily);

    if (newSettingsMap.get("forceDark") != null && !customSettings.forceDark.equals(newCustomSettings.forceDark)) {
      if (WebViewFeature.isFeatureSupported(WebViewFeature.FORCE_DARK))
        WebSettingsCompat.setForceDark(settings, newCustomSettings.forceDark);
      else if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q)
        settings.setForceDark(newCustomSettings.forceDark);
    }

    if (newSettingsMap.get("forceDarkStrategy") != null &&
            !customSettings.forceDarkStrategy.equals(newCustomSettings.forceDarkStrategy) &&
            WebViewFeature.isFeatureSupported(WebViewFeature.FORCE_DARK_STRATEGY)) {
      WebSettingsCompat.setForceDarkStrategy(settings, newCustomSettings.forceDarkStrategy);
    }

    if (newSettingsMap.get("geolocationEnabled") != null && customSettings.geolocationEnabled != newCustomSettings.geolocationEnabled)
      settings.setGeolocationEnabled(newCustomSettings.geolocationEnabled);

    if (newSettingsMap.get("layoutAlgorithm") != null && customSettings.layoutAlgorithm != newCustomSettings.layoutAlgorithm) {
      if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.KITKAT && newCustomSettings.layoutAlgorithm.equals(WebSettings.LayoutAlgorithm.TEXT_AUTOSIZING)) {
        settings.setLayoutAlgorithm(newCustomSettings.layoutAlgorithm);
      } else {
        settings.setLayoutAlgorithm(newCustomSettings.layoutAlgorithm);
      }
    }

    if (newSettingsMap.get("loadWithOverviewMode") != null && customSettings.loadWithOverviewMode != newCustomSettings.loadWithOverviewMode)
      settings.setLoadWithOverviewMode(newCustomSettings.loadWithOverviewMode);

    if (newSettingsMap.get("loadsImagesAutomatically") != null && customSettings.loadsImagesAutomatically != newCustomSettings.loadsImagesAutomatically)
      settings.setLoadsImagesAutomatically(newCustomSettings.loadsImagesAutomatically);

    if (newSettingsMap.get("minimumFontSize") != null && !customSettings.minimumFontSize.equals(newCustomSettings.minimumFontSize))
      settings.setMinimumFontSize(newCustomSettings.minimumFontSize);

    if (newSettingsMap.get("minimumLogicalFontSize") != null && !customSettings.minimumLogicalFontSize.equals(newCustomSettings.minimumLogicalFontSize))
      settings.setMinimumLogicalFontSize(newCustomSettings.minimumLogicalFontSize);

    if (newSettingsMap.get("initialScale") != null && !customSettings.initialScale.equals(newCustomSettings.initialScale))
      setInitialScale(newCustomSettings.initialScale);

    if (newSettingsMap.get("needInitialFocus") != null && customSettings.needInitialFocus != newCustomSettings.needInitialFocus)
      settings.setNeedInitialFocus(newCustomSettings.needInitialFocus);

    if (newSettingsMap.get("offscreenPreRaster") != null && customSettings.offscreenPreRaster != newCustomSettings.offscreenPreRaster) {
      if (WebViewFeature.isFeatureSupported(WebViewFeature.OFF_SCREEN_PRERASTER))
        WebSettingsCompat.setOffscreenPreRaster(settings, newCustomSettings.offscreenPreRaster);
      else if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M)
        settings.setOffscreenPreRaster(newCustomSettings.offscreenPreRaster);
    }

    if (newSettingsMap.get("sansSerifFontFamily") != null && !customSettings.sansSerifFontFamily.equals(newCustomSettings.sansSerifFontFamily))
      settings.setSansSerifFontFamily(newCustomSettings.sansSerifFontFamily);

    if (newSettingsMap.get("serifFontFamily") != null && !customSettings.serifFontFamily.equals(newCustomSettings.serifFontFamily))
      settings.setSerifFontFamily(newCustomSettings.serifFontFamily);

    if (newSettingsMap.get("standardFontFamily") != null && !customSettings.standardFontFamily.equals(newCustomSettings.standardFontFamily))
      settings.setStandardFontFamily(newCustomSettings.standardFontFamily);

    if (newSettingsMap.get("preferredContentMode") != null && !customSettings.preferredContentMode.equals(newCustomSettings.preferredContentMode)) {
      switch (fromValue(newCustomSettings.preferredContentMode)) {
        case DESKTOP:
          setDesktopMode(true);
          break;
        case MOBILE:
        case RECOMMENDED:
          setDesktopMode(false);
          break;
      }
    }

    if (newSettingsMap.get("saveFormData") != null && customSettings.saveFormData != newCustomSettings.saveFormData)
      settings.setSaveFormData(newCustomSettings.saveFormData);

    if (newSettingsMap.get("incognito") != null && customSettings.incognito != newCustomSettings.incognito)
      setIncognito(newCustomSettings.incognito);

    if (customSettings.useHybridComposition) {
      if (newSettingsMap.get("hardwareAcceleration") != null && customSettings.hardwareAcceleration != newCustomSettings.hardwareAcceleration) {
        if (newCustomSettings.hardwareAcceleration)
          setLayerType(View.LAYER_TYPE_HARDWARE, null);
        else
          setLayerType(View.LAYER_TYPE_NONE, null);
      }
    }

    if (newSettingsMap.get("regexToCancelSubFramesLoading") != null && (customSettings.regexToCancelSubFramesLoading == null ||
            !customSettings.regexToCancelSubFramesLoading.equals(newCustomSettings.regexToCancelSubFramesLoading))) {
      if (newCustomSettings.regexToCancelSubFramesLoading == null)
        regexToCancelSubFramesLoadingCompiled = null;
      else
        regexToCancelSubFramesLoadingCompiled = Pattern.compile(customSettings.regexToCancelSubFramesLoading);
    }

    if (newCustomSettings.contentBlockers != null) {
      contentBlockerHandler.getRuleList().clear();
      for (Map<String, Map<String, Object>> contentBlocker : newCustomSettings.contentBlockers) {
        // compile ContentBlockerTrigger urlFilter
        ContentBlockerTrigger trigger = ContentBlockerTrigger.fromMap(contentBlocker.get("trigger"));
        ContentBlockerAction action = ContentBlockerAction.fromMap(contentBlocker.get("action"));
        contentBlockerHandler.getRuleList().add(new ContentBlocker(trigger, action));
      }
    }

    if (newSettingsMap.get("scrollBarStyle") != null && !customSettings.scrollBarStyle.equals(newCustomSettings.scrollBarStyle))
      setScrollBarStyle(newCustomSettings.scrollBarStyle);

    if (newSettingsMap.get("scrollBarDefaultDelayBeforeFade") != null && (customSettings.scrollBarDefaultDelayBeforeFade == null ||
            !customSettings.scrollBarDefaultDelayBeforeFade.equals(newCustomSettings.scrollBarDefaultDelayBeforeFade)))
      setScrollBarDefaultDelayBeforeFade(newCustomSettings.scrollBarDefaultDelayBeforeFade);

    if (newSettingsMap.get("scrollbarFadingEnabled") != null && !customSettings.scrollbarFadingEnabled.equals(newCustomSettings.scrollbarFadingEnabled))
      setScrollbarFadingEnabled(newCustomSettings.scrollbarFadingEnabled);

    if (newSettingsMap.get("scrollBarFadeDuration") != null && (customSettings.scrollBarFadeDuration == null ||
            !customSettings.scrollBarFadeDuration.equals(newCustomSettings.scrollBarFadeDuration)))
      setScrollBarFadeDuration(newCustomSettings.scrollBarFadeDuration);

    if (newSettingsMap.get("verticalScrollbarPosition") != null && !customSettings.verticalScrollbarPosition.equals(newCustomSettings.verticalScrollbarPosition))
      setVerticalScrollbarPosition(newCustomSettings.verticalScrollbarPosition);

    if (newSettingsMap.get("disableVerticalScroll") != null && customSettings.disableVerticalScroll != newCustomSettings.disableVerticalScroll)
      setVerticalScrollBarEnabled(!newCustomSettings.disableVerticalScroll && newCustomSettings.verticalScrollBarEnabled);

    if (newSettingsMap.get("disableHorizontalScroll") != null && customSettings.disableHorizontalScroll != newCustomSettings.disableHorizontalScroll)
      setHorizontalScrollBarEnabled(!newCustomSettings.disableHorizontalScroll && newCustomSettings.horizontalScrollBarEnabled);

    if (newSettingsMap.get("overScrollMode") != null && !customSettings.overScrollMode.equals(newCustomSettings.overScrollMode))
      setOverScrollMode(newCustomSettings.overScrollMode);

    if (newSettingsMap.get("networkAvailable") != null && customSettings.networkAvailable != newCustomSettings.networkAvailable)
      setNetworkAvailable(newCustomSettings.networkAvailable);

    if (newSettingsMap.get("rendererPriorityPolicy") != null && (customSettings.rendererPriorityPolicy == null ||
            (customSettings.rendererPriorityPolicy.get("rendererRequestedPriority") != newCustomSettings.rendererPriorityPolicy.get("rendererRequestedPriority") ||
                    customSettings.rendererPriorityPolicy.get("waivedWhenNotVisible") != newCustomSettings.rendererPriorityPolicy.get("waivedWhenNotVisible"))) &&
            Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
      setRendererPriorityPolicy(
              (int) newCustomSettings.rendererPriorityPolicy.get("rendererRequestedPriority"),
              (boolean) newCustomSettings.rendererPriorityPolicy.get("waivedWhenNotVisible"));
    }

    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
      if (newSettingsMap.get("verticalScrollbarThumbColor") != null && !Util.objEquals(customSettings.verticalScrollbarThumbColor, newCustomSettings.verticalScrollbarThumbColor))
        setVerticalScrollbarThumbDrawable(new ColorDrawable(Color.parseColor(newCustomSettings.verticalScrollbarThumbColor)));

      if (newSettingsMap.get("verticalScrollbarTrackColor") != null && !Util.objEquals(customSettings.verticalScrollbarTrackColor, newCustomSettings.verticalScrollbarTrackColor))
        setVerticalScrollbarTrackDrawable(new ColorDrawable(Color.parseColor(newCustomSettings.verticalScrollbarTrackColor)));

      if (newSettingsMap.get("horizontalScrollbarThumbColor") != null && !Util.objEquals(customSettings.horizontalScrollbarThumbColor, newCustomSettings.horizontalScrollbarThumbColor))
        setHorizontalScrollbarThumbDrawable(new ColorDrawable(Color.parseColor(newCustomSettings.horizontalScrollbarThumbColor)));

      if (newSettingsMap.get("horizontalScrollbarTrackColor") != null && !Util.objEquals(customSettings.horizontalScrollbarTrackColor, newCustomSettings.horizontalScrollbarTrackColor))
        setHorizontalScrollbarTrackDrawable(new ColorDrawable(Color.parseColor(newCustomSettings.horizontalScrollbarTrackColor)));
    }

    if (newSettingsMap.get("algorithmicDarkeningAllowed") != null &&
            !Util.objEquals(customSettings.algorithmicDarkeningAllowed, newCustomSettings.algorithmicDarkeningAllowed) &&
            WebViewFeature.isFeatureSupported(WebViewFeature.ALGORITHMIC_DARKENING) && Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
      WebSettingsCompat.setAlgorithmicDarkeningAllowed(settings, newCustomSettings.algorithmicDarkeningAllowed);
    }
    if (newSettingsMap.get("enterpriseAuthenticationAppLinkPolicyEnabled") != null &&
            !Util.objEquals(customSettings.enterpriseAuthenticationAppLinkPolicyEnabled, newCustomSettings.enterpriseAuthenticationAppLinkPolicyEnabled) &&
            WebViewFeature.isFeatureSupported(WebViewFeature.ENTERPRISE_AUTHENTICATION_APP_LINK_POLICY)) {
      WebSettingsCompat.setEnterpriseAuthenticationAppLinkPolicyEnabled(settings, newCustomSettings.enterpriseAuthenticationAppLinkPolicyEnabled);
    }
    if (newSettingsMap.get("requestedWithHeaderOriginAllowList") != null &&
            !Util.objEquals(customSettings.requestedWithHeaderOriginAllowList, newCustomSettings.requestedWithHeaderOriginAllowList) &&
            WebViewFeature.isFeatureSupported(WebViewFeature.REQUESTED_WITH_HEADER_ALLOW_LIST)) {
      WebSettingsCompat.setRequestedWithHeaderOriginAllowList(settings, newCustomSettings.requestedWithHeaderOriginAllowList);
    }

    if (plugin != null) {
      if (webViewAssetLoaderExt != null) {
        webViewAssetLoaderExt.dispose();
      }
      webViewAssetLoaderExt = WebViewAssetLoaderExt.fromMap(customSettings.webViewAssetLoader, plugin, getContext());
    }

    customSettings = newCustomSettings;
  }

  public Map<String, Object> getCustomSettings() {
    return (customSettings != null) ? customSettings.getRealSettings(this) : null;
  }

  public void enablePluginScriptAtRuntime(final String flagVariable,
                                          final boolean enable,
                                          final PluginScript pluginScript) {
    evaluateJavascript("window." + flagVariable, null, new ValueCallback<String>() {
      @Override
      public void onReceiveValue(String value) {
        boolean alreadyLoaded = value != null && !value.equalsIgnoreCase("null");
        if (alreadyLoaded) {
          String enableSource = "window." + flagVariable + " = " + enable + ";";
          evaluateJavascript(enableSource, null, null);
          if (!enable) {
            userContentController.removePluginScript(pluginScript);
          }
        } else if (enable) {
          evaluateJavascript(pluginScript.getSource(), null, null);
          userContentController.addPluginScript(pluginScript);
        }
      }
    });
  }

  public void injectDeferredObject(String source, @Nullable final ContentWorld contentWorld, String jsWrapper, @Nullable final ValueCallback<String> resultCallback) {
    final String resultUuid = contentWorld != null && !contentWorld.equals(ContentWorld.PAGE) ? UUID.randomUUID().toString() : null;
    String scriptToInject = source;
    if (jsWrapper != null) {
      org.json.JSONArray jsonEsc = new org.json.JSONArray();
      jsonEsc.put(source);
      String jsonRepr = jsonEsc.toString();
      String jsonSourceString = jsonRepr.substring(1, jsonRepr.length() - 1);
      scriptToInject = String.format(jsWrapper, jsonSourceString);
    }
    if (resultUuid != null && resultCallback != null) {
      evaluateJavaScriptContentWorldCallbacks.put(resultUuid, resultCallback);
      scriptToInject = Util.replaceAll(PluginScriptsUtil.EVALUATE_JAVASCRIPT_WITH_CONTENT_WORLD_WRAPPER_JS_SOURCE,
              PluginScriptsUtil.VAR_RANDOM_NAME, "_" + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + "_" + Math.round(Math.random() * 1000000))
              .replace(PluginScriptsUtil.VAR_PLACEHOLDER_VALUE, UserContentController.escapeCode(source))
              .replace(PluginScriptsUtil.VAR_RESULT_UUID, resultUuid);
    }
    final String finalScriptToInject = scriptToInject;
    mainLooperHandler.post(new Runnable() {
      @Override
      public void run() {
        String scriptToInject = userContentController.generateCodeForScriptEvaluation(finalScriptToInject, contentWorld);
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.KITKAT) {
          // This action will have the side-effect of blurring the currently focused element
          loadUrl("javascript:" + scriptToInject.replaceAll("[\r\n]+", ""));
          if (contentWorld != null && resultCallback != null) {
            resultCallback.onReceiveValue("");
          }
        } else {
          evaluateJavascript(scriptToInject, new ValueCallback<String>() {
            @Override
            public void onReceiveValue(String s) {
              if (resultUuid != null || resultCallback == null)
                return;
              resultCallback.onReceiveValue(s);
            }
          });
        }
      }
    });
  }

  public void evaluateJavascript(String source, @Nullable ContentWorld contentWorld, @Nullable ValueCallback<String> resultCallback) {
    injectDeferredObject(source, contentWorld, null, resultCallback);
  }

  public void injectJavascriptFileFromUrl(String urlFile, @Nullable Map<String, Object> scriptHtmlTagAttributes) {
    String scriptAttributes = "";
    if (scriptHtmlTagAttributes != null) {
      String typeAttr = (String) scriptHtmlTagAttributes.get("type");
      if (typeAttr != null) {
        scriptAttributes += " script.type = '" + typeAttr.replaceAll("'", "\\\\'") + "'; ";
      }
      String idAttr = (String) scriptHtmlTagAttributes.get("id");
      if (idAttr != null) {
        String scriptIdEscaped = idAttr.replaceAll("'", "\\\\'");
        scriptAttributes += " script.id = '" + scriptIdEscaped + "'; ";
        scriptAttributes += " script.onload = function() {" +
        "  if (window." + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + " != null) {" +
        "    window." + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + ".callHandler('onInjectedScriptLoaded', '" + scriptIdEscaped + "');" +
        "  }" +
        "};";
        scriptAttributes += " script.onerror = function() {" +
        "  if (window." + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + " != null) {" +
        "    window." + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + ".callHandler('onInjectedScriptError', '" + scriptIdEscaped + "');" +
        "  }" +
        "};";
      }
      Boolean asyncAttr = (Boolean) scriptHtmlTagAttributes.get("async");
      if (asyncAttr != null && asyncAttr) {
        scriptAttributes += " script.async = true; ";
      }
      Boolean deferAttr = (Boolean) scriptHtmlTagAttributes.get("defer");
      if (deferAttr != null && deferAttr) {
        scriptAttributes += " script.defer = true; ";
      }
      String crossOriginAttr = (String) scriptHtmlTagAttributes.get("crossOrigin");
      if (crossOriginAttr != null) {
        scriptAttributes += " script.crossOrigin = '" + crossOriginAttr.replaceAll("'", "\\\\'") + "'; ";
      }
      String integrityAttr = (String) scriptHtmlTagAttributes.get("integrity");
      if (integrityAttr != null) {
        scriptAttributes += " script.integrity = '" + integrityAttr.replaceAll("'", "\\\\'") + "'; ";
      }
      Boolean noModuleAttr = (Boolean) scriptHtmlTagAttributes.get("noModule");
      if (noModuleAttr != null && noModuleAttr) {
        scriptAttributes += " script.noModule = true; ";
      }
      String nonceAttr = (String) scriptHtmlTagAttributes.get("nonce");
      if (nonceAttr != null) {
        scriptAttributes += " script.nonce = '" + nonceAttr.replaceAll("'", "\\\\'") + "'; ";
      }
      String referrerPolicyAttr = (String) scriptHtmlTagAttributes.get("referrerPolicy");
      if (referrerPolicyAttr != null) {
        scriptAttributes += " script.referrerPolicy = '" + referrerPolicyAttr.replaceAll("'", "\\\\'") + "'; ";
      }
    }
    String jsWrapper = "(function(d) { var script = d.createElement('script'); " + scriptAttributes +
            " script.src = %s; if (d.body != null) { d.body.appendChild(script); } })(document);";
    injectDeferredObject(urlFile, null, jsWrapper, null);
  }

  public void injectCSSCode(String source) {
    String jsWrapper = "(function(d) { var style = d.createElement('style'); style.innerHTML = %s;" +
            " if (d.head != null) { d.head.appendChild(style); } })(document);";
    injectDeferredObject(source, null, jsWrapper, null);
  }

  public void injectCSSFileFromUrl(String urlFile, @Nullable Map<String, Object> cssLinkHtmlTagAttributes) {
    String cssLinkAttributes = "";
    String alternateStylesheet = "";
    if (cssLinkHtmlTagAttributes != null) {
      String idAttr = (String) cssLinkHtmlTagAttributes.get("id");
      if (idAttr != null) {
        cssLinkAttributes += " link.id = '" + idAttr.replaceAll("'", "\\\\'") + "'; ";
      }
      String mediaAttr = (String) cssLinkHtmlTagAttributes.get("media");
      if (mediaAttr != null) {
        cssLinkAttributes += " link.media = '" + mediaAttr.replaceAll("'", "\\\\'") + "'; ";
      }
      String crossOriginAttr = (String) cssLinkHtmlTagAttributes.get("crossOrigin");
      if (crossOriginAttr != null) {
        cssLinkAttributes += " link.crossOrigin = '" + crossOriginAttr.replaceAll("'", "\\\\'") + "'; ";
      }
      String integrityAttr = (String) cssLinkHtmlTagAttributes.get("integrity");
      if (integrityAttr != null) {
        cssLinkAttributes += " link.integrity = '" + integrityAttr.replaceAll("'", "\\\\'") + "'; ";
      }
      String referrerPolicyAttr = (String) cssLinkHtmlTagAttributes.get("referrerPolicy");
      if (referrerPolicyAttr != null) {
        cssLinkAttributes += " link.referrerPolicy = '" + referrerPolicyAttr.replaceAll("'", "\\\\'") + "'; ";
      }
      Boolean disabledAttr = (Boolean) cssLinkHtmlTagAttributes.get("disabled");
      if (disabledAttr != null && disabledAttr) {
        cssLinkAttributes += " link.disabled = true; ";
      }
      Boolean alternateAttr = (Boolean) cssLinkHtmlTagAttributes.get("alternate");
      if (alternateAttr != null && alternateAttr) {
        alternateStylesheet = "alternate ";
      }
      String titleAttr = (String) cssLinkHtmlTagAttributes.get("title");
      if (titleAttr != null) {
        cssLinkAttributes += " link.title = '" + titleAttr.replaceAll("'", "\\\\'") + "'; ";
      }
    }
    String jsWrapper = "(function(d) { var link = d.createElement('link'); link.rel='" + alternateStylesheet + "stylesheet'; link.type='text/css'; " +
            cssLinkAttributes + " link.href = %s; if (d.head != null) { d.head.appendChild(link); } })(document);";
    injectDeferredObject(urlFile, null, jsWrapper, null);
  }

  public HashMap<String, Object> getCopyBackForwardList() {
    WebBackForwardList currentList = copyBackForwardList();
    int currentSize = currentList.getSize();
    int currentIndex = currentList.getCurrentIndex();

    List<HashMap<String, String>> history = new ArrayList<HashMap<String, String>>();

    for (int i = 0; i < currentSize; i++) {
      WebHistoryItem historyItem = currentList.getItemAtIndex(i);
      HashMap<String, String> historyItemMap = new HashMap<>();

      historyItemMap.put("originalUrl", historyItem.getOriginalUrl());
      historyItemMap.put("title", historyItem.getTitle());
      historyItemMap.put("url", historyItem.getUrl());

      history.add(historyItemMap);
    }

    HashMap<String, Object> result = new HashMap<>();

    result.put("list", history);
    result.put("currentIndex", currentIndex);

    return result;
  }

  @Override
  protected void onScrollChanged(int x,
                                 int y,
                                 int oldX,
                                 int oldY) {
    super.onScrollChanged(x, y, oldX, oldY);

    if (floatingContextMenu != null) {
      floatingContextMenu.setAlpha(0f);
      floatingContextMenu.setVisibility(View.GONE);
    }

    if (channelDelegate != null) channelDelegate.onScrollChanged(x, y);
  }

  public void scrollTo(Integer x, Integer y, Boolean animated) {
    if (animated) {
      PropertyValuesHolder pvhX = PropertyValuesHolder.ofInt("scrollX", x);
      PropertyValuesHolder pvhY = PropertyValuesHolder.ofInt("scrollY", y);
      ObjectAnimator anim = ObjectAnimator.ofPropertyValuesHolder(this, pvhX, pvhY);
      anim.setDuration(300).start();
    } else {
      scrollTo(x, y);
    }
  }

  public void scrollBy(Integer x, Integer y, Boolean animated) {
    if (animated) {
      PropertyValuesHolder pvhX = PropertyValuesHolder.ofInt("scrollX", getScrollX() + x);
      PropertyValuesHolder pvhY = PropertyValuesHolder.ofInt("scrollY", getScrollY() + y);
      ObjectAnimator anim = ObjectAnimator.ofPropertyValuesHolder(this, pvhX, pvhY);
      anim.setDuration(300).start();
    } else {
      scrollBy(x, y);
    }
  }

  class DownloadStartListener implements DownloadListener {
    @Override
    public void onDownloadStart(String url, String userAgent, String contentDisposition, String mimeType, long contentLength) {
      DownloadStartRequest downloadStartRequest = new DownloadStartRequest(
        url,
        userAgent,
        contentDisposition,
        mimeType,
        contentLength,
        URLUtil.guessFileName(url, contentDisposition, mimeType),
        null
      );
      if (channelDelegate != null) channelDelegate.onDownloadStartRequest(downloadStartRequest);
    }
  }

  public void setDesktopMode(final boolean enabled) {
    final WebSettings webSettings = getSettings();

    final String newUserAgent;
    if (enabled) {
      newUserAgent = webSettings.getUserAgentString().replace("Mobile", "eliboM").replace("Android", "diordnA");
    } else {
      newUserAgent = webSettings.getUserAgentString().replace("eliboM", "Mobile").replace("diordnA", "Android");
    }

    webSettings.setUserAgentString(newUserAgent);
    webSettings.setUseWideViewPort(enabled);
    webSettings.setLoadWithOverviewMode(enabled);
    webSettings.setSupportZoom(enabled);
    webSettings.setBuiltInZoomControls(enabled);
  }

  @RequiresApi(api = Build.VERSION_CODES.KITKAT)
  @Nullable
  public String printCurrentPage(@Nullable PrintJobSettings settings) {
    if (plugin != null && plugin.activity != null) {
      // Get a PrintManager instance
      PrintManager printManager = (PrintManager) plugin.activity.getSystemService(Context.PRINT_SERVICE);

      if (printManager != null) {
        PrintAttributes.Builder builder = new PrintAttributes.Builder();

        String jobName = (getTitle() != null ? getTitle() : getUrl()) + " Document";

        if (settings != null) {
          if (settings.jobName != null && !settings.jobName.isEmpty()) {
            jobName = settings.jobName;
          }
          if (settings.orientation != null) {
            int orientation = settings.orientation;
            switch (orientation) {
              case 0:
                // PORTRAIT
                builder.setMediaSize(PrintAttributes.MediaSize.UNKNOWN_PORTRAIT);
                break;
              case 1:
                // LANDSCAPE
                builder.setMediaSize(PrintAttributes.MediaSize.UNKNOWN_LANDSCAPE);
                break;
            }
          }
//          if (settings.margins != null) {
//            // for some reason, Android doesn't set the margins
//            builder.setMinMargins(settings.margins.toMargins());
//          }
          if (settings.mediaSize != null) {
            builder.setMediaSize(settings.mediaSize.toMediaSize());
          }
          if (settings.colorMode != null) {
            builder.setColorMode(settings.colorMode);
          }
          if (settings.duplexMode != null && Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            builder.setDuplexMode(settings.duplexMode);
          }
          if (settings.resolution != null) {
            builder.setResolution(settings.resolution.toResolution());
          }
        }

        // Get a printCurrentPage adapter instance
        PrintDocumentAdapter printAdapter;
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
          printAdapter = createPrintDocumentAdapter(jobName);
        } else {
          printAdapter = createPrintDocumentAdapter();
        }

        // Create a printCurrentPage job with name and adapter instance
        android.print.PrintJob job = printManager.print(jobName, printAdapter, builder.build());

        if (settings != null && settings.handledByClient && plugin.printJobManager != null) {
          String id = UUID.randomUUID().toString();
          PrintJobController printJobController = new PrintJobController(id, job, settings, plugin);
          plugin.printJobManager.jobs.put(printJobController.id, printJobController);
          return id;
        }
      } else {
        Log.e(LOG_TAG, "No PrintManager available");
      }
    }
    return null;
  }

  @Override
  public void onCreateContextMenu(ContextMenu menu) {
    super.onCreateContextMenu(menu);
    sendOnCreateContextMenuEvent();
  }

  private void sendOnCreateContextMenuEvent() {
    com.pichillilorenzo.flutter_inappwebview_android.types.HitTestResult hitTestResult =
            com.pichillilorenzo.flutter_inappwebview_android.types.HitTestResult.fromWebViewHitTestResult(getHitTestResult());
    if (channelDelegate != null) channelDelegate.onCreateContextMenu(hitTestResult);
  }

  private Point contextMenuPoint = new Point(0, 0);
  private Point lastTouch = new Point(0, 0);

  @Override
  public boolean onTouchEvent(MotionEvent ev) {
    lastTouch = new Point((int) ev.getX(), (int) ev.getY());

    ViewParent parent = getParent();
    if (parent instanceof PullToRefreshLayout) {
      PullToRefreshLayout pullToRefreshLayout = (PullToRefreshLayout) parent;
      if (ev.getActionMasked() == MotionEvent.ACTION_DOWN) {
        pullToRefreshLayout.setEnabled(false);
      }
    }

    return super.onTouchEvent(ev);
  }

  @Override
  protected void onOverScrolled(int scrollX, int scrollY, boolean clampedX, boolean clampedY) {
    super.onOverScrolled(scrollX, scrollY, clampedX, clampedY);

    boolean overScrolledHorizontally = canScrollHorizontally() && clampedX;
    boolean overScrolledVertically = canScrollVertically() && clampedY;

    ViewParent parent = getParent();
    if (parent instanceof PullToRefreshLayout && overScrolledVertically && scrollY <= 10) {
      PullToRefreshLayout pullToRefreshLayout = (PullToRefreshLayout) parent;
      // change over scroll mode to OVER_SCROLL_NEVER in order to disable temporarily the glow effect
      setOverScrollMode(OVER_SCROLL_NEVER);
      pullToRefreshLayout.setEnabled(pullToRefreshLayout.settings.enabled);
      // reset over scroll mode
      setOverScrollMode(customSettings.overScrollMode);
    }

    if (overScrolledHorizontally || overScrolledVertically) {
      if (channelDelegate != null) channelDelegate.onOverScrolled(scrollX, scrollY, overScrolledHorizontally, overScrolledVertically);
    }
  }

  @Override
  public boolean dispatchTouchEvent(MotionEvent event) {
    return super.dispatchTouchEvent(event);
  }

  @Override
  public InputConnection onCreateInputConnection(EditorInfo outAttrs) {
    InputConnection connection = super.onCreateInputConnection(outAttrs);
    if (connection == null && !customSettings.useHybridComposition && containerView != null) {
      // workaround to hide the Keyboard when the user click outside
      // on something not focusable such as input or a textarea.
      containerView
              .getHandler()
              .postDelayed(
                      new Runnable() {
                        @Override
                        public void run() {
                          InputMethodManager imm =
                                  (InputMethodManager) getContext().getSystemService(INPUT_METHOD_SERVICE);

                          boolean isAcceptingText = false;
                          if (imm != null) {
                            try {
                              // imm.isAcceptingText() seems to sometimes crash on some devices
                              isAcceptingText = imm.isAcceptingText();
                            } catch (Exception ignored) {
                            }
                          }

                          if (containerView != null && imm != null && !isAcceptingText) {
                            imm.hideSoftInputFromWindow(
                                    containerView.getWindowToken(), InputMethodManager.HIDE_NOT_ALWAYS);
                          }
                        }
                      },
                      128);
    }
    return connection;
  }

  @Override
  public ActionMode startActionMode(ActionMode.Callback callback) {
    if (customSettings.useHybridComposition && !customSettings.disableContextMenu && (contextMenu == null || contextMenu.keySet().size() == 0)) {
      return super.startActionMode(callback);
    }
    return rebuildActionMode(super.startActionMode(callback), callback);
  }

  @RequiresApi(api = Build.VERSION_CODES.M)
  @Override
  public ActionMode startActionMode(ActionMode.Callback callback, int type) {
    if (customSettings.useHybridComposition && !customSettings.disableContextMenu && (contextMenu == null || contextMenu.keySet().size() == 0)) {
      return super.startActionMode(callback, type);
    }
    return rebuildActionMode(super.startActionMode(callback, type), callback);
  }

  public ActionMode rebuildActionMode(
          final ActionMode actionMode,
          final ActionMode.Callback callback
  ) {
    // fix Android 10 clipboard not working properly https://github.com/pichillilorenzo/flutter_inappwebview/issues/678
    if (!customSettings.useHybridComposition && containerView != null) {
      onWindowFocusChanged(containerView.isFocused());
    }

    boolean hasBeenRemovedAndRebuilt = false;
    if (floatingContextMenu != null) {
      hideContextMenu();
      hasBeenRemovedAndRebuilt = true;
    }
    if (actionMode == null) {
      return null;
    }

    Menu actionMenu = actionMode.getMenu();
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
      actionMode.hide(3000);
    }
    List<MenuItem> defaultMenuItems = new ArrayList<>();
    for (int i = 0; i < actionMenu.size(); i++) {
      defaultMenuItems.add(actionMenu.getItem(i));
    }
    actionMenu.clear();
    actionMode.finish();
    if (customSettings.disableContextMenu) {
      return actionMode;
    }

    floatingContextMenu = (LinearLayout) LayoutInflater.from(this.getContext())
            .inflate(R.layout.floating_action_mode, this, false);
    HorizontalScrollView horizontalScrollView = (HorizontalScrollView) floatingContextMenu.getChildAt(0);
    LinearLayout menuItemListLayout = (LinearLayout) horizontalScrollView.getChildAt(0);

    List<Map<String, Object>> customMenuItems = new ArrayList<>();
    ContextMenuSettings contextMenuSettings = new ContextMenuSettings();
    if (contextMenu != null) {
      customMenuItems = (List<Map<String, Object>>) contextMenu.get("menuItems");
      Map<String, Object> contextMenuSettingsMap = (Map<String, Object>) contextMenu.get("settings");
      if (contextMenuSettingsMap != null) {
        contextMenuSettings.parse(contextMenuSettingsMap);
      }
    }
    customMenuItems = customMenuItems == null ? new ArrayList<Map<String, Object>>() : customMenuItems;

    if (contextMenuSettings.hideDefaultSystemContextMenuItems == null || !contextMenuSettings.hideDefaultSystemContextMenuItems) {
      for (final MenuItem menuItem : defaultMenuItems) {
        final int itemId = menuItem.getItemId();
        final String itemTitle = menuItem.getTitle().toString();

        TextView text = (TextView) LayoutInflater.from(this.getContext())
                .inflate(R.layout.floating_action_mode_item, this, false);
        text.setText(itemTitle);
        text.setOnClickListener(new View.OnClickListener() {
          @Override
          public void onClick(View v) {
            hideContextMenu();
            callback.onActionItemClicked(actionMode, menuItem);

            if (channelDelegate != null) channelDelegate.onContextMenuActionItemClicked(itemId, itemTitle);
          }
        });
        if (floatingContextMenu != null) {
          menuItemListLayout.addView(text);
        }
      }
    }

    for (final Map<String, Object> menuItem : customMenuItems) {
      final int itemId = (int) menuItem.get("id");
      final String itemTitle = (String) menuItem.get("title");
      TextView text = (TextView) LayoutInflater.from(this.getContext())
              .inflate(R.layout.floating_action_mode_item, this, false);
      text.setText(itemTitle);
      text.setOnClickListener(new OnClickListener() {
        @Override
        public void onClick(View v) {
          hideContextMenu();

          if (channelDelegate != null) channelDelegate.onContextMenuActionItemClicked(itemId, itemTitle);
        }
      });
      if (floatingContextMenu != null) {
        menuItemListLayout.addView(text);
      }
    }

    final int x = (lastTouch != null) ? lastTouch.x : 0;
    final int y = (lastTouch != null) ? lastTouch.y : 0;
    contextMenuPoint = new Point(x, y);

    if (floatingContextMenu != null) {
      floatingContextMenu.getViewTreeObserver().addOnGlobalLayoutListener(new ViewTreeObserver.OnGlobalLayoutListener() {

        @Override
        public void onGlobalLayout() {
          if (floatingContextMenu != null) {
            floatingContextMenu.getViewTreeObserver().removeOnGlobalLayoutListener(this);
            if (getSettings().getJavaScriptEnabled()) {
              onScrollStopped();
            } else {
              onFloatingActionGlobalLayout(x, y);
            }
          }
        }
      });
      addView(floatingContextMenu, new LayoutParams(ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT, x, y));
      if (hasBeenRemovedAndRebuilt) {
        sendOnCreateContextMenuEvent();
      }
      if (checkContextMenuShouldBeClosedTask != null) {
        checkContextMenuShouldBeClosedTask.run();
      }
    }

    return actionMode;
  }

  public void onFloatingActionGlobalLayout(int x, int y) {
    int maxWidth = getWidth();
    int maxHeight = getHeight();
    int width = floatingContextMenu.getWidth();
    int height = floatingContextMenu.getHeight();
    int curx = x - (width / 2);
    if (curx < 0) {
      curx = 0;
    } else if (curx + width > maxWidth) {
      curx = maxWidth - width;
    }
    // float size = 12 * scale;
    float cury = y - (height * 1.5f);
    if (cury < 0) {
      cury = y + height;
    }

    updateViewLayout(
            floatingContextMenu,
            new LayoutParams(ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT, curx + getScrollX(), ((int) cury) + getScrollY())
    );

    mainLooperHandler.post(new Runnable() {
      @Override
      public void run() {
        if (floatingContextMenu != null) {
          floatingContextMenu.setVisibility(View.VISIBLE);
          floatingContextMenu.animate().alpha(1f).setDuration(100).setListener(null);
        }
      }
    });
  }

  public void hideContextMenu() {
    removeView(floatingContextMenu);
    floatingContextMenu = null;

    if (channelDelegate != null) channelDelegate.onHideContextMenu();
  }

  public void onScrollStopped() {
    if (floatingContextMenu != null && Build.VERSION.SDK_INT >= Build.VERSION_CODES.KITKAT) {
      adjustFloatingContextMenuPosition();
    }
  }

  @RequiresApi(api = Build.VERSION_CODES.KITKAT)
  public void adjustFloatingContextMenuPosition() {
    evaluateJavascript("(function(){" +
            "  var selection = window.getSelection();" +
            "  var rangeY = null;" +
            "  if (selection != null && selection.rangeCount > 0) {" +
            "    var range = selection.getRangeAt(0);" +
            "    var clientRect = range.getClientRects();" +
            "    if (clientRect.length > 0) {" +
            "      rangeY = clientRect[0].y;" +
            "    } else if (document.activeElement != null && document.activeElement.tagName.toLowerCase() !== 'iframe') {" +
            "      var boundingClientRect = document.activeElement.getBoundingClientRect();" +
            "      rangeY = boundingClientRect.y;" +
            "    }" +
            "  }" +
            "  return rangeY;" +
            "})();", new ValueCallback<String>() {
      @Override
      public void onReceiveValue(String value) {
        if (floatingContextMenu != null) {
          if (value != null && !value.equalsIgnoreCase("null")) {
            int x = contextMenuPoint.x;
            int y = (int) ((Float.parseFloat(value) * Util.getPixelDensity(getContext())) + (floatingContextMenu.getHeight() / 3.5));
            contextMenuPoint.y = y;
            onFloatingActionGlobalLayout(x, y);
          } else {
            floatingContextMenu.setVisibility(View.VISIBLE);
            floatingContextMenu.animate().alpha(1f).setDuration(100).setListener(null);
            onFloatingActionGlobalLayout(contextMenuPoint.x, contextMenuPoint.y);
          }
        }
      }
    });
  }

  @RequiresApi(api = Build.VERSION_CODES.KITKAT)
  public void getSelectedText(final ValueCallback<String> resultCallback) {
    evaluateJavascript(PluginScriptsUtil.GET_SELECTED_TEXT_JS_SOURCE, new ValueCallback<String>() {
      @Override
      public void onReceiveValue(String value) {
        value = (value != null && !value.equalsIgnoreCase("null")) ? value.substring(1, value.length() - 1) : null;
        resultCallback.onReceiveValue(value);
      }
    });
  }

  public Map<String, Object> requestFocusNodeHref() {
    Message msg = InAppWebView.mHandler.obtainMessage();
    requestFocusNodeHref(msg);
    Bundle bundle = msg.peekData();

    Map<String, Object> obj = new HashMap<>();
    obj.put("src", bundle.getString("src"));
    obj.put("url", bundle.getString("url"));
    obj.put("title", bundle.getString("title"));

    return obj;
  }

  public Map<String, Object> requestImageRef() {
    Message msg = InAppWebView.mHandler.obtainMessage();
    requestImageRef(msg);
    Bundle bundle = msg.peekData();

    Map<String, Object> obj = new HashMap<>();
    obj.put("url", bundle.getString("url"));

    return obj;
  }

  @RequiresApi(api = Build.VERSION_CODES.LOLLIPOP)
  public void callAsyncJavaScript(String functionBody, Map<String, Object> arguments, @Nullable ContentWorld contentWorld, @Nullable ValueCallback<String> resultCallback) {
    String resultUuid = UUID.randomUUID().toString();
    if (resultCallback != null) {
      callAsyncJavaScriptCallbacks.put(resultUuid, resultCallback);
    }

    JSONObject functionArguments = new JSONObject(arguments);
    Iterator<String> keys = functionArguments.keys();

    List<String> functionArgumentNamesList = new ArrayList<>();
    List<String> functionArgumentValuesList = new ArrayList<>();
    while (keys.hasNext()) {
      String key = keys.next();
      functionArgumentNamesList.add(key);
      functionArgumentValuesList.add("obj." + key);
    }

    String functionArgumentNames = TextUtils.join(", ", functionArgumentNamesList);
    String functionArgumentValues = TextUtils.join(", ", functionArgumentValuesList);
    String functionArgumentsObj = Util.JSONStringify(arguments);

    String sourceToInject = PluginScriptsUtil.CALL_ASYNC_JAVA_SCRIPT_WRAPPER_JS_SOURCE
            .replace(PluginScriptsUtil.VAR_FUNCTION_ARGUMENT_NAMES, functionArgumentNames)
            .replace(PluginScriptsUtil.VAR_FUNCTION_ARGUMENT_VALUES, functionArgumentValues)
            .replace(PluginScriptsUtil.VAR_FUNCTION_ARGUMENTS_OBJ, functionArgumentsObj)
            .replace(PluginScriptsUtil.VAR_FUNCTION_BODY, functionBody)
            .replace(PluginScriptsUtil.VAR_RESULT_UUID, resultUuid)
            .replace(PluginScriptsUtil.VAR_RESULT_UUID, resultUuid);

    sourceToInject = userContentController.generateCodeForScriptEvaluation(sourceToInject, contentWorld);
    evaluateJavascript(sourceToInject, null);
  }

  @TargetApi(Build.VERSION_CODES.LOLLIPOP)
  public void isSecureContext(final ValueCallback<Boolean> resultCallback) {
    evaluateJavascript("window.isSecureContext", new ValueCallback<String>() {
      @Override
      public void onReceiveValue(String value) {
        if (value == null || value.isEmpty() || value.equalsIgnoreCase("null")
                || value.equalsIgnoreCase("false")) {
          resultCallback.onReceiveValue(false);
          return;
        }
        resultCallback.onReceiveValue(true);
      }
    });
  }

  public boolean canScrollVertically() {
    return computeVerticalScrollRange() > computeVerticalScrollExtent();
  }

  public boolean canScrollHorizontally() {
    return computeHorizontalScrollRange() > computeHorizontalScrollExtent();
  }

  public WebMessageChannel createCompatWebMessageChannel() {
    String id = UUID.randomUUID().toString();
    WebMessageChannel webMessageChannel = new WebMessageChannel(id, this);
    webMessageChannels.put(id, webMessageChannel);
    return webMessageChannel;
  }

  @Override
  public WebMessageChannel createWebMessageChannel(ValueCallback<WebMessageChannel> callback) {
    WebMessageChannel webMessageChannel = createCompatWebMessageChannel();
    callback.onReceiveValue(webMessageChannel);
    return webMessageChannel;
  }

  public void addWebMessageListener(@NonNull WebMessageListener webMessageListener) throws Exception {
    if (WebViewFeature.isFeatureSupported(WebViewFeature.WEB_MESSAGE_LISTENER)) {
      WebViewCompat.addWebMessageListener(this, webMessageListener.jsObjectName, webMessageListener.allowedOriginRules, webMessageListener.listener);
      webMessageListeners.add(webMessageListener);
    }
  }

  public void disposeWebMessageChannels() {
    for (WebMessageChannel webMessageChannel : webMessageChannels.values()) {
      webMessageChannel.dispose();
    }
    webMessageChannels.clear();
  }

  public void disposeWebMessageListeners() {
    for (WebMessageListener webMessageListener : webMessageListeners) {
      webMessageListener.dispose();
    }
    webMessageListeners.clear();
  }

  @Override
  public Looper getWebViewLooper() {
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
      return super.getWebViewLooper();
    }
    return Looper.getMainLooper();
  }

  @Override
  public boolean isInFullscreen() {
    return inFullscreen;
  }

  @Override
  public void setInFullscreen(boolean inFullscreen) {
    this.inFullscreen = inFullscreen;
  }

  @Override
  public void postWebMessage(com.pichillilorenzo.flutter_inappwebview_android.types.WebMessage message, Uri targetOrigin, ValueCallback<String> callback) throws Exception {
    throw new UnsupportedOperationException();
  }

  @Override
  protected void onWindowVisibilityChanged(int visibility) {
    if (customSettings.allowBackgroundAudioPlaying) {
      if (visibility != View.GONE) {
        super.onWindowVisibilityChanged(View.VISIBLE);
      }
      return;
    }
    super.onWindowVisibilityChanged(visibility);
  }

  public float getZoomScale() {
    return zoomScale;
  }

  @Override
  public void getZoomScale(final ValueCallback<Float> callback) {
    callback.onReceiveValue(zoomScale);
  }

  @Nullable
  public Map<String, Object> getContextMenu() {
    return contextMenu;
  }

  public void setContextMenu(@Nullable Map<String, Object> contextMenu) {
    this.contextMenu = contextMenu;
  }

  @Nullable
  public InAppWebViewFlutterPlugin getPlugin() {
    return plugin;
  }

  public void setPlugin(@Nullable InAppWebViewFlutterPlugin plugin) {
    this.plugin = plugin;
  }

  @Nullable
  public InAppBrowserDelegate getInAppBrowserDelegate() {
    return inAppBrowserDelegate;
  }

  public void setInAppBrowserDelegate(@Nullable InAppBrowserDelegate inAppBrowserDelegate) {
    this.inAppBrowserDelegate = inAppBrowserDelegate;
  }

  public UserContentController getUserContentController() {
    return userContentController;
  }

  public void setUserContentController(UserContentController userContentController) {
    this.userContentController = userContentController;
  }

  public Map<String, WebMessageChannel> getWebMessageChannels() {
    return webMessageChannels;
  }

  public void setWebMessageChannels(Map<String, WebMessageChannel> webMessageChannels) {
    this.webMessageChannels = webMessageChannels;
  }

  @Override
  public void getContentHeight(ValueCallback<Integer> callback) {
    callback.onReceiveValue(getContentHeight());
  }

  public void getContentWidth(final ValueCallback<Integer> callback) {
    evaluateJavascript("document.documentElement.scrollWidth;", new ValueCallback<String>() {
      @Override
      public void onReceiveValue(@Nullable String value) {
        Integer contentWidth = null;
        if (value != null && !value.equalsIgnoreCase("null")) {
          contentWidth = Integer.parseInt(value);
        }
        callback.onReceiveValue(contentWidth);
      }
    });
  }

  @Override
  public void getHitTestResult(ValueCallback<com.pichillilorenzo.flutter_inappwebview_android.types.HitTestResult> callback) {
    callback.onReceiveValue(com.pichillilorenzo.flutter_inappwebview_android.types.HitTestResult.fromWebViewHitTestResult(getHitTestResult()));
  }

  @Nullable
  @Override
  public WebViewChannelDelegate getChannelDelegate() {
    return channelDelegate;
  }

  @Override
  public void setChannelDelegate(@Nullable WebViewChannelDelegate channelDelegate) {
    this.channelDelegate = channelDelegate;
  }

  @Override
  public void dispose() {
    if (channelDelegate != null) {
      channelDelegate.dispose();
      channelDelegate = null;
    }
    super.dispose();
    WebSettings settings = getSettings();
    settings.setJavaScriptEnabled(false);
    removeJavascriptInterface(JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME);
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q && WebViewFeature.isFeatureSupported(WebViewFeature.WEB_VIEW_RENDERER_CLIENT_BASIC_USAGE)) {
      WebViewCompat.setWebViewRenderProcessClient(this, null);
    }
    setWebChromeClient(new WebChromeClient());
    setWebViewClient(new WebViewClient() {
      public void onPageFinished(WebView view, String url) {
        destroy();
      }
    });
    interceptOnlyAsyncAjaxRequestsPluginScript = null;
    userContentController.dispose();
    if (findInteractionController != null) {
      findInteractionController.dispose();
      findInteractionController = null;
    }
    if (webViewAssetLoaderExt != null) {
      webViewAssetLoaderExt.dispose();
      webViewAssetLoaderExt = null;
    }
    if (windowId != null && plugin != null && plugin.inAppWebViewManager != null) {
      plugin.inAppWebViewManager.windowWebViewMessages.remove(windowId);
    }
    mainLooperHandler.removeCallbacksAndMessages(null);
    mHandler.removeCallbacksAndMessages(null);
    disposeWebMessageChannels();
    disposeWebMessageListeners();
    removeAllViews();
    if (checkContextMenuShouldBeClosedTask != null)
      removeCallbacks(checkContextMenuShouldBeClosedTask);
    if (checkScrollStoppedTask != null)
      removeCallbacks(checkScrollStoppedTask);
    callAsyncJavaScriptCallbacks.clear();
    evaluateJavaScriptContentWorldCallbacks.clear();
    inAppBrowserDelegate = null;
    if (inAppWebViewRenderProcessClient != null) {
      inAppWebViewRenderProcessClient.dispose();
      inAppWebViewRenderProcessClient = null;
    }
    if (inAppWebViewChromeClient != null) {
      inAppWebViewChromeClient.dispose();
      inAppWebViewChromeClient = null;
    }
    if (inAppWebViewClientCompat != null) {
      inAppWebViewClientCompat.dispose();
      inAppWebViewClientCompat = null;
    }
    if (inAppWebViewClient != null) {
      inAppWebViewClient.dispose();
      inAppWebViewClient = null;
    }
    if (javaScriptBridgeInterface != null) {
      javaScriptBridgeInterface.dispose();
      javaScriptBridgeInterface = null;
    }
    plugin = null;
    loadUrl("about:blank");
  }

  @Override
  public void destroy() {
    super.destroy();
  }
}
