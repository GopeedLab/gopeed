package com.pichillilorenzo.flutter_inappwebview_android.webview;

import android.content.Context;
import android.net.Uri;
import android.net.http.SslCertificate;
import android.os.Looper;
import android.webkit.ValueCallback;
import android.webkit.WebMessage;
import android.webkit.WebView;

import androidx.annotation.Nullable;

import com.pichillilorenzo.flutter_inappwebview_android.InAppWebViewFlutterPlugin;
import com.pichillilorenzo.flutter_inappwebview_android.in_app_browser.InAppBrowserDelegate;
import com.pichillilorenzo.flutter_inappwebview_android.print_job.PrintJobSettings;
import com.pichillilorenzo.flutter_inappwebview_android.types.ContentWorld;
import com.pichillilorenzo.flutter_inappwebview_android.types.HitTestResult;
import com.pichillilorenzo.flutter_inappwebview_android.types.URLRequest;
import com.pichillilorenzo.flutter_inappwebview_android.types.UserContentController;
import com.pichillilorenzo.flutter_inappwebview_android.webview.in_app_webview.InAppWebViewSettings;
import com.pichillilorenzo.flutter_inappwebview_android.webview.web_message.WebMessageChannel;
import com.pichillilorenzo.flutter_inappwebview_android.webview.web_message.WebMessageListener;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;

import io.flutter.plugin.common.MethodChannel;

public interface InAppWebViewInterface {
  Context getContext();
  String getUrl();
  String getTitle();
  int getProgress();
  void loadUrl(URLRequest urlRequest);
  void postUrl(String url, byte[] postData);
  void loadDataWithBaseURL(String baseUrl, String data,
                           String mimeType, String encoding, String historyUrl);
  void loadFile(String assetFilePath) throws IOException;
  void evaluateJavascript(String source, ContentWorld contentWorld, ValueCallback<String> resultCallback);
  void injectJavascriptFileFromUrl(String urlFile, Map<String, Object> scriptHtmlTagAttributes);
  void injectCSSCode(String source);
  void injectCSSFileFromUrl(String urlFile, Map<String, Object> cssLinkHtmlTagAttributes);
  void reload();
  void goBack();
  boolean canGoBack();
  void goForward();
  boolean canGoForward();
  void goBackOrForward(int steps);
  boolean canGoBackOrForward(int steps);
  void stopLoading();
  boolean isLoading();
  void takeScreenshot(Map<String, Object> screenshotConfiguration, MethodChannel.Result result);
  void setSettings(InAppWebViewSettings newSettings, HashMap<String, Object> newSettingsMap);
  Map<String, Object> getCustomSettings();
  HashMap<String, Object> getCopyBackForwardList();
  void clearAllCache();
  void clearSslPreferences();
  void findAllAsync(String find);
  void findNext(boolean forward);
  void clearMatches();
  void scrollTo(Integer x, Integer y, Boolean animated);
  void scrollBy(Integer x, Integer y, Boolean animated);
  void onPause();
  void onResume();
  void pauseTimers();
  void resumeTimers();
  @Nullable
  String printCurrentPage(@Nullable PrintJobSettings settings);
  int getContentHeight();
  void getContentHeight(ValueCallback<Integer> callback);
  void getContentWidth(ValueCallback<Integer> callback);
  void zoomBy(float zoomFactor);
  String getOriginalUrl();
  void getSelectedText(ValueCallback<String> callback);
  WebView.HitTestResult getHitTestResult();
  void getHitTestResult(ValueCallback<HitTestResult> callback);
  boolean pageDown(boolean bottom);
  boolean pageUp(boolean top);
  void saveWebArchive(String basename, boolean autoname, ValueCallback<String> callback);
  boolean zoomIn();
  boolean zoomOut();
  void clearFocus();
  Map<String, Object> requestFocusNodeHref();
  Map<String, Object> requestImageRef();
  int getScrollX();
  int getScrollY();
  SslCertificate getCertificate();
  void clearHistory();
  void callAsyncJavaScript(String functionBody, Map<String, Object> arguments, ContentWorld contentWorld, ValueCallback<String> resultCallback);
  void isSecureContext(final ValueCallback<Boolean> resultCallback);
  WebMessageChannel createCompatWebMessageChannel();
  WebMessageChannel createWebMessageChannel(ValueCallback<WebMessageChannel> callback);
  void postWebMessage(WebMessage message, Uri targetOrigin);
  void postWebMessage(com.pichillilorenzo.flutter_inappwebview_android.types.WebMessage message, Uri targetOrigin, ValueCallback<String> callback) throws Exception;
  void addWebMessageListener(WebMessageListener webMessageListener) throws Exception;
  boolean canScrollVertically();
  boolean canScrollHorizontally();
  float getZoomScale();
  void getZoomScale(ValueCallback<Float> callback);
  Map<String, Object> getContextMenu();
  void setContextMenu(Map<String, Object> contextMenu);
  InAppWebViewFlutterPlugin getPlugin();
  void setPlugin(InAppWebViewFlutterPlugin plugin);
  InAppBrowserDelegate getInAppBrowserDelegate();
  void setInAppBrowserDelegate(InAppBrowserDelegate inAppBrowserDelegate);
  UserContentController getUserContentController();
  void setUserContentController(UserContentController userContentController);
  Map<String, WebMessageChannel> getWebMessageChannels();
  void setWebMessageChannels(Map<String, WebMessageChannel> webMessageChannels);
  void disposeWebMessageChannels();
  void disposeWebMessageListeners();
  Looper getWebViewLooper();
  boolean isInFullscreen();
  void setInFullscreen(boolean inFullscreen);
  @Nullable
  WebViewChannelDelegate getChannelDelegate();
  void setChannelDelegate(@Nullable WebViewChannelDelegate eventWebViewChannelDelegate);
}
