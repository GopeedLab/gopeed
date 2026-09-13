package com.pichillilorenzo.flutter_inappwebview_android.webview;

import android.net.Uri;
import android.os.Build;
import android.webkit.ValueCallback;
import android.webkit.WebView;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.webkit.WebMessageCompat;
import androidx.webkit.WebMessagePortCompat;
import androidx.webkit.WebViewCompat;
import androidx.webkit.WebViewFeature;

import com.pichillilorenzo.flutter_inappwebview_android.Util;
import com.pichillilorenzo.flutter_inappwebview_android.find_interaction.FindInteractionChannelDelegate;
import com.pichillilorenzo.flutter_inappwebview_android.in_app_browser.InAppBrowserActivity;
import com.pichillilorenzo.flutter_inappwebview_android.in_app_browser.InAppBrowserSettings;
import com.pichillilorenzo.flutter_inappwebview_android.print_job.PrintJobSettings;
import com.pichillilorenzo.flutter_inappwebview_android.types.BaseCallbackResultImpl;
import com.pichillilorenzo.flutter_inappwebview_android.types.ChannelDelegateImpl;
import com.pichillilorenzo.flutter_inappwebview_android.types.ClientCertChallenge;
import com.pichillilorenzo.flutter_inappwebview_android.types.ClientCertResponse;
import com.pichillilorenzo.flutter_inappwebview_android.types.ContentWorld;
import com.pichillilorenzo.flutter_inappwebview_android.types.CreateWindowAction;
import com.pichillilorenzo.flutter_inappwebview_android.types.CustomSchemeResponse;
import com.pichillilorenzo.flutter_inappwebview_android.types.DownloadStartRequest;
import com.pichillilorenzo.flutter_inappwebview_android.types.GeolocationPermissionShowPromptResponse;
import com.pichillilorenzo.flutter_inappwebview_android.types.HitTestResult;
import com.pichillilorenzo.flutter_inappwebview_android.types.HttpAuthResponse;
import com.pichillilorenzo.flutter_inappwebview_android.types.HttpAuthenticationChallenge;
import com.pichillilorenzo.flutter_inappwebview_android.types.JsAlertResponse;
import com.pichillilorenzo.flutter_inappwebview_android.types.JsBeforeUnloadResponse;
import com.pichillilorenzo.flutter_inappwebview_android.types.JsConfirmResponse;
import com.pichillilorenzo.flutter_inappwebview_android.types.JsPromptResponse;
import com.pichillilorenzo.flutter_inappwebview_android.types.NavigationAction;
import com.pichillilorenzo.flutter_inappwebview_android.types.NavigationActionPolicy;
import com.pichillilorenzo.flutter_inappwebview_android.types.PermissionResponse;
import com.pichillilorenzo.flutter_inappwebview_android.types.SafeBrowsingResponse;
import com.pichillilorenzo.flutter_inappwebview_android.types.ServerTrustAuthResponse;
import com.pichillilorenzo.flutter_inappwebview_android.types.ServerTrustChallenge;
import com.pichillilorenzo.flutter_inappwebview_android.types.SslCertificateExt;
import com.pichillilorenzo.flutter_inappwebview_android.types.SyncBaseCallbackResultImpl;
import com.pichillilorenzo.flutter_inappwebview_android.types.URLRequest;
import com.pichillilorenzo.flutter_inappwebview_android.types.UserScript;
import com.pichillilorenzo.flutter_inappwebview_android.types.WebMessageCompatExt;
import com.pichillilorenzo.flutter_inappwebview_android.types.WebMessagePortCompatExt;
import com.pichillilorenzo.flutter_inappwebview_android.types.WebResourceErrorExt;
import com.pichillilorenzo.flutter_inappwebview_android.types.WebResourceRequestExt;
import com.pichillilorenzo.flutter_inappwebview_android.types.WebResourceResponseExt;
import com.pichillilorenzo.flutter_inappwebview_android.webview.in_app_webview.InAppWebView;
import com.pichillilorenzo.flutter_inappwebview_android.webview.in_app_webview.InAppWebViewSettings;
import com.pichillilorenzo.flutter_inappwebview_android.webview.web_message.WebMessageChannel;
import com.pichillilorenzo.flutter_inappwebview_android.webview.web_message.WebMessageListener;

import java.io.IOException;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import io.flutter.plugin.common.MethodCall;
import io.flutter.plugin.common.MethodChannel;

public class WebViewChannelDelegate extends ChannelDelegateImpl {
  static final String LOG_TAG = "WebViewChannelDelegate";

  @Nullable
  private InAppWebView webView;

  public WebViewChannelDelegate(@NonNull InAppWebView webView, @NonNull MethodChannel channel) {
    super(channel);
    this.webView = webView;
  }

  @Override
  public void onMethodCall(@NonNull MethodCall call, @NonNull final MethodChannel.Result result) {
    WebViewChannelDelegateMethods method = null;
    try {
      method = WebViewChannelDelegateMethods.valueOf(call.method);
    } catch (IllegalArgumentException e) {
      result.notImplemented();
      return;
    }
    switch (method) {
      case getUrl:
        result.success((webView != null) ? webView.getUrl() : null);
        break;
      case getTitle:
        result.success((webView != null) ? webView.getTitle() : null);
        break;
      case getProgress:
        result.success((webView != null) ? webView.getProgress() : null);
        break;
      case loadUrl:
        if (webView != null) {
          Map<String, Object> urlRequest = (Map<String, Object>) call.argument("urlRequest");
          webView.loadUrl(URLRequest.fromMap(urlRequest));
        }
        result.success(true);
        break;
      case postUrl:
        if (webView != null) {
          String url = (String) call.argument("url");
          byte[] postData = (byte[]) call.argument("postData");
          webView.postUrl(url, postData);
        }
        result.success(true);
        break;
      case loadData:
        if (webView != null) {
          String data = (String) call.argument("data");
          String mimeType = (String) call.argument("mimeType");
          String encoding = (String) call.argument("encoding");
          String baseUrl = (String) call.argument("baseUrl");
          String historyUrl = (String) call.argument("historyUrl");
          webView.loadDataWithBaseURL(baseUrl, data, mimeType, encoding, historyUrl);
        }
        result.success(true);
        break;
      case loadFile:
        if (webView != null) {
          String assetFilePath = (String) call.argument("assetFilePath");
          try {
            webView.loadFile(assetFilePath);
          } catch (IOException e) {
            e.printStackTrace();
            result.error(LOG_TAG, e.getMessage(), null);
            return;
          }
        }
        result.success(true);
        break;
      case evaluateJavascript:
        if (webView != null) {
          String source = (String) call.argument("source");
          Map<String, Object> contentWorldMap = (Map<String, Object>) call.argument("contentWorld");
          ContentWorld contentWorld = ContentWorld.fromMap(contentWorldMap);
          webView.evaluateJavascript(source, contentWorld, new ValueCallback<String>() {
            @Override
            public void onReceiveValue(String value) {
              result.success(value);
            }
          });
        } else {
          result.success(null);
        }
        break;
      case injectJavascriptFileFromUrl:
        if (webView != null) {
          String urlFile = (String) call.argument("urlFile");
          Map<String, Object> scriptHtmlTagAttributes = (Map<String, Object>) call.argument("scriptHtmlTagAttributes");
          webView.injectJavascriptFileFromUrl(urlFile, scriptHtmlTagAttributes);
        }
        result.success(true);
        break;
      case injectCSSCode:
        if (webView != null) {
          String source = (String) call.argument("source");
          webView.injectCSSCode(source);
        }
        result.success(true);
        break;
      case injectCSSFileFromUrl:
        if (webView != null) {
          String urlFile = (String) call.argument("urlFile");
          Map<String, Object> cssLinkHtmlTagAttributes = (Map<String, Object>) call.argument("cssLinkHtmlTagAttributes");
          webView.injectCSSFileFromUrl(urlFile, cssLinkHtmlTagAttributes);
        }
        result.success(true);
        break;
      case reload:
        if (webView != null)
          webView.reload();
        result.success(true);
        break;
      case goBack:
        if (webView != null)
          webView.goBack();
        result.success(true);
        break;
      case canGoBack:
        result.success((webView != null) && webView.canGoBack());
        break;
      case goForward:
        if (webView != null)
          webView.goForward();
        result.success(true);
        break;
      case canGoForward:
        result.success((webView != null) && webView.canGoForward());
        break;
      case goBackOrForward:
        if (webView != null)
          webView.goBackOrForward((Integer) call.argument("steps"));
        result.success(true);
        break;
      case canGoBackOrForward:
        result.success((webView != null) && webView.canGoBackOrForward((Integer) call.argument("steps")));
        break;
      case stopLoading:
        if (webView != null)
          webView.stopLoading();
        result.success(true);
        break;
      case isLoading:
        result.success((webView != null) && webView.isLoading());
        break;
      case takeScreenshot:
        if (webView != null) {
          Map<String, Object> screenshotConfiguration = (Map<String, Object>) call.argument("screenshotConfiguration");
          webView.takeScreenshot(screenshotConfiguration, result);
        } else
          result.success(null);
        break;
      case setSettings:
        if (webView != null && webView.getInAppBrowserDelegate() instanceof InAppBrowserActivity) {
          InAppBrowserActivity inAppBrowserActivity = (InAppBrowserActivity) webView.getInAppBrowserDelegate();
          InAppBrowserSettings inAppBrowserSettings = new InAppBrowserSettings();
          HashMap<String, Object> inAppBrowserSettingsMap = (HashMap<String, Object>) call.argument("settings");
          inAppBrowserSettings.parse(inAppBrowserSettingsMap);
          inAppBrowserActivity.setSettings(inAppBrowserSettings, inAppBrowserSettingsMap);
        } else if (webView != null) {
          InAppWebViewSettings inAppWebViewSettings = new InAppWebViewSettings();
          HashMap<String, Object> inAppWebViewSettingsMap = (HashMap<String, Object>) call.argument("settings");
          inAppWebViewSettings.parse(inAppWebViewSettingsMap);
          webView.setSettings(inAppWebViewSettings, inAppWebViewSettingsMap);
        }
        result.success(true);
        break;
      case getSettings:
        if (webView != null && webView.getInAppBrowserDelegate() instanceof InAppBrowserActivity) {
          InAppBrowserActivity inAppBrowserActivity = (InAppBrowserActivity) webView.getInAppBrowserDelegate();
          result.success(inAppBrowserActivity.getCustomSettings());
        } else {
          result.success((webView != null) ? webView.getCustomSettings() : null);
        }
        break;
      case close:
        if (webView != null && webView.getInAppBrowserDelegate() instanceof InAppBrowserActivity) {
          InAppBrowserActivity inAppBrowserActivity = (InAppBrowserActivity) webView.getInAppBrowserDelegate();
          inAppBrowserActivity.close(result);
        } else {
          result.notImplemented();
        }
        break;
      case show:
        if (webView != null && webView.getInAppBrowserDelegate() instanceof InAppBrowserActivity) {
          InAppBrowserActivity inAppBrowserActivity = (InAppBrowserActivity) webView.getInAppBrowserDelegate();
          inAppBrowserActivity.show();
          result.success(true);
        } else {
          result.notImplemented();
        }
        break;
      case hide:
        if (webView != null && webView.getInAppBrowserDelegate() instanceof InAppBrowserActivity) {
          InAppBrowserActivity inAppBrowserActivity = (InAppBrowserActivity) webView.getInAppBrowserDelegate();
          inAppBrowserActivity.hide();
          result.success(true);
        } else {
          result.notImplemented();
        }
        break;
      case isHidden:
        if (webView != null && webView.getInAppBrowserDelegate() instanceof InAppBrowserActivity) {
          InAppBrowserActivity inAppBrowserActivity = (InAppBrowserActivity) webView.getInAppBrowserDelegate();
          result.success(inAppBrowserActivity.isHidden);
        } else {
          result.notImplemented();
        }
        break;
      case getCopyBackForwardList:
        result.success((webView != null) ? webView.getCopyBackForwardList() : null);
        break;
      case startSafeBrowsing:
        if (webView != null && WebViewFeature.isFeatureSupported(WebViewFeature.START_SAFE_BROWSING)) {
          WebViewCompat.startSafeBrowsing(webView.getContext(), new ValueCallback<Boolean>() {
            @Override
            public void onReceiveValue(Boolean success) {
              result.success(success);
            }
          });
        } else {
          result.success(false);
        }
        break;
      case clearCache:
        if (webView != null)
          webView.clearAllCache();
        result.success(true);
        break;
      case clearSslPreferences:
        if (webView != null)
          webView.clearSslPreferences();
        result.success(true);
        break;
      case findAll:
        if (webView != null) {
          String find = (String) call.argument("find");
          webView.findAllAsync(find);
        }
        result.success(true);
        break;
      case findNext:
        if (webView != null) {
          Boolean forward = (Boolean) call.argument("forward");
          webView.findNext(forward);
        }
        result.success(true);
        break;
      case clearMatches:
        if (webView != null) {
          webView.clearMatches();
        }
        result.success(true);
        break;
      case scrollTo:
        if (webView != null) {
          Integer x = (Integer) call.argument("x");
          Integer y = (Integer) call.argument("y");
          Boolean animated = (Boolean) call.argument("animated");
          webView.scrollTo(x, y, animated);
        }
        result.success(true);
        break;
      case scrollBy:
        if (webView != null) {
          Integer x = (Integer) call.argument("x");
          Integer y = (Integer) call.argument("y");
          Boolean animated = (Boolean) call.argument("animated");
          webView.scrollBy(x, y, animated);
        }
        result.success(true);
        break;
      case pause:
        if (webView != null) {
          webView.onPause();
        }
        result.success(true);
        break;
      case resume:
        if (webView != null) {
          webView.onResume();
        }
        result.success(true);
        break;
      case pauseTimers:
        if (webView != null) {
          webView.pauseTimers();
        }
        result.success(true);
        break;
      case resumeTimers:
        if (webView != null) {
          webView.resumeTimers();
        }
        result.success(true);
        break;
      case printCurrentPage:
        if (webView != null && Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
          PrintJobSettings settings = new PrintJobSettings();
          Map<String, Object> settingsMap = (Map<String, Object>) call.argument("settings");
          if (settingsMap != null) {
            settings.parse(settingsMap);
          }
          result.success(webView.printCurrentPage(settings));
        } else {
          result.success(null);
        }
        break;
      case getContentHeight:
        if (webView instanceof InAppWebView) {
          result.success(webView.getContentHeight());
        } else {
          result.success(null);
        }
        break;
      case getContentWidth:
        if (webView instanceof InAppWebView) {
          webView.getContentWidth(new ValueCallback<Integer>() {
            @Override
            public void onReceiveValue(@Nullable Integer contentWidth) {
              result.success(contentWidth);
            }
          });
        } else {
          result.success(null);
        }
        break;
      case zoomBy:
        if (webView != null && Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
          double zoomFactor = (double) call.argument("zoomFactor");
          webView.zoomBy((float) zoomFactor);
        }
        result.success(true);
        break;
      case getOriginalUrl:
        result.success((webView != null) ? webView.getOriginalUrl() : null);
        break;
      case getZoomScale:
        if (webView instanceof InAppWebView) {
          result.success(webView.getZoomScale());
        } else {
          result.success(null);
        }
        break;
      case getSelectedText:
        if ((webView instanceof InAppWebView && Build.VERSION.SDK_INT >= Build.VERSION_CODES.KITKAT)) {
          webView.getSelectedText(new ValueCallback<String>() {
            @Override
            public void onReceiveValue(String value) {
              result.success(value);
            }
          });
        } else {
          result.success(null);
        }
        break;
      case getHitTestResult:
        if (webView instanceof InAppWebView) {
          result.success(HitTestResult.fromWebViewHitTestResult(webView.getHitTestResult()).toMap());
        } else {
          result.success(null);
        }
        break;
      case pageDown:
        if (webView != null) {
          boolean bottom = (boolean) call.argument("bottom");
          result.success(webView.pageDown(bottom));
        } else {
          result.success(false);
        }
        break;
      case pageUp:
        if (webView != null) {
          boolean top = (boolean) call.argument("top");
          result.success(webView.pageUp(top));
        } else {
          result.success(false);
        }
        break;
      case saveWebArchive:
        if (webView != null) {
          String filePath = (String) call.argument("filePath");
          boolean autoname = (boolean) call.argument("autoname");
          webView.saveWebArchive(filePath, autoname, new ValueCallback<String>() {
            @Override
            public void onReceiveValue(String value) {
              result.success(value);
            }
          });
        } else {
          result.success(null);
        }
        break;
      case zoomIn:
        if (webView != null) {
          result.success(webView.zoomIn());
        } else {
          result.success(false);
        }
        break;
      case zoomOut:
        if (webView != null) {
          result.success(webView.zoomOut());
        } else {
          result.success(false);
        }
        break;
      case clearFocus:
        if (webView != null) {
          webView.clearFocus();
        }
        result.success(true);
        break;
      case setContextMenu:
        if (webView != null) {
          Map<String, Object> contextMenu = (Map<String, Object>) call.argument("contextMenu");
          webView.setContextMenu(contextMenu);
        }
        result.success(true);
        break;
      case requestFocusNodeHref:
        if (webView != null) {
          result.success(webView.requestFocusNodeHref());
        } else {
          result.success(null);
        }
        break;
      case requestImageRef:
        if (webView != null) {
          result.success(webView.requestImageRef());
        } else {
          result.success(null);
        }
        break;
      case getScrollX:
        if (webView != null) {
          result.success(webView.getScrollX());
        } else {
          result.success(null);
        }
        break;
      case getScrollY:
        if (webView != null) {
          result.success(webView.getScrollY());
        } else {
          result.success(null);
        }
        break;
      case getCertificate:
        if (webView != null) {
          result.success(SslCertificateExt.toMap(webView.getCertificate()));
        } else {
          result.success(null);
        }
        break;
      case clearHistory:
        if (webView != null) {
          webView.clearHistory();
        }
        result.success(true);
        break;
      case addUserScript:
        if (webView != null && webView.getUserContentController() != null) {
          Map<String, Object> userScriptMap = (Map<String, Object>) call.argument("userScript");
          UserScript userScript = UserScript.fromMap(userScriptMap);
          result.success(webView.getUserContentController().addUserOnlyScript(userScript));
        } else {
          result.success(false);
        }
        break;
      case removeUserScript:
        if (webView != null && webView.getUserContentController() != null) {
          Integer index = (Integer) call.argument("index");
          Map<String, Object> userScriptMap = (Map<String, Object>) call.argument("userScript");
          UserScript userScript = UserScript.fromMap(userScriptMap);
          result.success(webView.getUserContentController().removeUserOnlyScriptAt(index, userScript.getInjectionTime()));
        } else {
          result.success(false);
        }
        break;
      case removeUserScriptsByGroupName:
        if (webView != null && webView.getUserContentController() != null) {
          String groupName = (String) call.argument("groupName");
          webView.getUserContentController().removeUserOnlyScriptsByGroupName(groupName);
        }
        result.success(true);
        break;
      case removeAllUserScripts:
        if (webView != null && webView.getUserContentController() != null) {
          webView.getUserContentController().removeAllUserOnlyScripts();
        }
        result.success(true);
        break;
      case callAsyncJavaScript:
        if (webView != null && Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
          String functionBody = (String) call.argument("functionBody");
          Map<String, Object> functionArguments = (Map<String, Object>) call.argument("arguments");
          Map<String, Object> contentWorldMap = (Map<String, Object>) call.argument("contentWorld");
          ContentWorld contentWorld = ContentWorld.fromMap(contentWorldMap);
          webView.callAsyncJavaScript(functionBody, functionArguments, contentWorld, new ValueCallback<String>() {
            @Override
            public void onReceiveValue(String value) {
              result.success(value);
            }
          });
        } else {
          result.success(null);
        }
        break;
      case isSecureContext:
        if (webView != null && Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
          webView.isSecureContext(new ValueCallback<Boolean>() {
            @Override
            public void onReceiveValue(Boolean value) {
              result.success(value);
            }
          });
        } else {
          result.success(false);
        }
        break;
      case createWebMessageChannel:
        if (webView != null) {
          if (webView instanceof InAppWebView && WebViewFeature.isFeatureSupported(WebViewFeature.CREATE_WEB_MESSAGE_CHANNEL)) {
            result.success(webView.createCompatWebMessageChannel().toMap());
          } else {
            result.success(null);
          }
        } else {
          result.success(null);
        }
        break;
      case postWebMessage:
        if (webView != null && WebViewFeature.isFeatureSupported(WebViewFeature.POST_WEB_MESSAGE)) {
          WebMessageCompatExt message = WebMessageCompatExt.fromMap((Map<String, Object>) call.argument("message"));
          String targetOrigin = (String) call.argument("targetOrigin");
          List<WebMessagePortCompat> compatPorts = new ArrayList<>();
          List<WebMessagePortCompatExt> portsExt = message.getPorts();
          if (portsExt != null) {
            for (WebMessagePortCompatExt portExt : portsExt) {
              WebMessageChannel webMessageChannel = webView.getWebMessageChannels().get(portExt.getWebMessageChannelId());
              if (webMessageChannel != null) {
                if (webView instanceof InAppWebView) {
                  compatPorts.add(webMessageChannel.compatPorts.get(portExt.getIndex()));
                }
              }
            }
          }
          Object data = message.getData();
          if (webView instanceof InAppWebView) {
            try {
              if (WebViewFeature.isFeatureSupported(WebViewFeature.WEB_MESSAGE_ARRAY_BUFFER) && data != null &&
                      message.getType() == WebMessageCompat.TYPE_ARRAY_BUFFER) {
                WebViewCompat.postWebMessage((WebView) webView,
                        new WebMessageCompat((byte[]) data, compatPorts.toArray(new WebMessagePortCompat[0])),
                        Uri.parse(targetOrigin));
              } else {
                WebViewCompat.postWebMessage((WebView) webView,
                        new WebMessageCompat(data != null ? data.toString() : null, compatPorts.toArray(new WebMessagePortCompat[0])),
                        Uri.parse(targetOrigin));
              }
              result.success(true);
            } catch (Exception e) {
              result.error(LOG_TAG, e.getMessage(), null);
            }
          }
        } else {
          result.success(true);
        }
        break;
      case addWebMessageListener:
        if (webView != null) {
          Map<String, Object> webMessageListenerMap = (Map<String, Object>) call.argument("webMessageListener");
          WebMessageListener webMessageListener = WebMessageListener.fromMap(webView, webView.getPlugin().messenger, webMessageListenerMap);
          if (webView instanceof InAppWebView && WebViewFeature.isFeatureSupported(WebViewFeature.WEB_MESSAGE_LISTENER)) {
            try {
              webView.addWebMessageListener(webMessageListener);
              result.success(true);
            } catch (Exception e) {
              result.error(LOG_TAG, e.getMessage(), null);
            }
          } else {
            result.success(true);
          }
        } else {
          result.success(true);
        }
        break;
      case canScrollVertically:
        if (webView != null) {
          result.success(webView.canScrollVertically());
        } else {
          result.success(false);
        }
        break;
      case canScrollHorizontally:
        if (webView != null) {
          result.success(webView.canScrollHorizontally());
        } else {
          result.success(false);
        }
        break;
      case isInFullscreen:
        if (webView != null) {
          result.success(webView.isInFullscreen());
        } else {
          result.success(false);
        }
        break;
      case clearFormData:
        if (webView != null) {
          webView.clearFormData();
        }
        result.success(true);
    }
  }

  /**
   * @deprecated Use {@link FindInteractionChannelDelegate#onFindResultReceived} instead.
   */
  @Deprecated
  public void onFindResultReceived(int activeMatchOrdinal, int numberOfMatches, boolean isDoneCounting) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("activeMatchOrdinal", activeMatchOrdinal);
    obj.put("numberOfMatches", numberOfMatches);
    obj.put("isDoneCounting", isDoneCounting);
    channel.invokeMethod("onFindResultReceived", obj);
  }

  public void onLongPressHitTestResult(HitTestResult hitTestResult) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    channel.invokeMethod("onLongPressHitTestResult", hitTestResult.toMap());
  }

  public void onScrollChanged(int x, int y) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("x", x);
    obj.put("y", y);
    channel.invokeMethod("onScrollChanged", obj);
  }

  public void onDownloadStartRequest(DownloadStartRequest downloadStartRequest) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    channel.invokeMethod("onDownloadStartRequest", downloadStartRequest.toMap());
  }

  public void onCreateContextMenu(HitTestResult hitTestResult) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    channel.invokeMethod("onCreateContextMenu", hitTestResult.toMap());
  }

  public void onOverScrolled(int scrollX, int scrollY, boolean clampedX, boolean clampedY) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("x", scrollX);
    obj.put("y", scrollY);
    obj.put("clampedX", clampedX);
    obj.put("clampedY", clampedY);
    channel.invokeMethod("onOverScrolled", obj);
  }

  public void onContextMenuActionItemClicked(int itemId, String itemTitle) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("id", itemId);
    obj.put("androidId", itemId);
    obj.put("iosId", null);
    obj.put("title", itemTitle);
    channel.invokeMethod("onContextMenuActionItemClicked", obj);
  }

  public void onHideContextMenu() {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    channel.invokeMethod("onHideContextMenu", obj);
  }

  public void onEnterFullscreen() {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    channel.invokeMethod("onEnterFullscreen", obj);
  }

  public void onExitFullscreen() {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    channel.invokeMethod("onExitFullscreen", obj);
  }

  public static class JsAlertCallback extends BaseCallbackResultImpl<JsAlertResponse> {
    @Nullable
    @Override
    public JsAlertResponse decodeResult(@Nullable Object obj) {
      return JsAlertResponse.fromMap((Map<String, Object>) obj);
    }
  }

  public void onJsAlert(String url, String message, Boolean isMainFrame, @NonNull JsAlertCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) {
      callback.defaultBehaviour(null);
      return;
    }
    Map<String, Object> obj = new HashMap<>();
    obj.put("url", url);
    obj.put("message", message);
    obj.put("isMainFrame", isMainFrame);
    channel.invokeMethod("onJsAlert", obj, callback);
  }

  public static class JsConfirmCallback extends BaseCallbackResultImpl<JsConfirmResponse> {
    @Nullable
    @Override
    public JsConfirmResponse decodeResult(@Nullable Object obj) {
      return JsConfirmResponse.fromMap((Map<String, Object>) obj);
    }
  }

  public void onJsConfirm(String url, String message, Boolean isMainFrame, @NonNull JsConfirmCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) {
      callback.defaultBehaviour(null);
      return;
    }
    Map<String, Object> obj = new HashMap<>();
    obj.put("url", url);
    obj.put("message", message);
    obj.put("isMainFrame", isMainFrame);
    channel.invokeMethod("onJsConfirm", obj, callback);
  }

  public static class JsPromptCallback extends BaseCallbackResultImpl<JsPromptResponse> {
    @Nullable
    @Override
    public JsPromptResponse decodeResult(@Nullable Object obj) {
      return JsPromptResponse.fromMap((Map<String, Object>) obj);
    }
  }

  public void onJsPrompt(String url, String message, String defaultValue, Boolean isMainFrame, @NonNull JsPromptCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) {
      callback.defaultBehaviour(null);
      return;
    }
    Map<String, Object> obj = new HashMap<>();
    obj.put("url", url);
    obj.put("message", message);
    obj.put("defaultValue", defaultValue);
    obj.put("isMainFrame", isMainFrame);
    channel.invokeMethod("onJsPrompt", obj, callback);
  }

  public static class JsBeforeUnloadCallback extends BaseCallbackResultImpl<JsBeforeUnloadResponse> {
    @Nullable
    @Override
    public JsBeforeUnloadResponse decodeResult(@Nullable Object obj) {
      return JsBeforeUnloadResponse.fromMap((Map<String, Object>) obj);
    }
  }

  public void onJsBeforeUnload(String url, String message, @NonNull JsBeforeUnloadCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) {
      callback.defaultBehaviour(null);
      return;
    }
    Map<String, Object> obj = new HashMap<>();
    obj.put("url", url);
    obj.put("message", message);
    channel.invokeMethod("onJsBeforeUnload", obj, callback);
  }

  public static class CreateWindowCallback extends BaseCallbackResultImpl<Boolean> {
    @Nullable
    @Override
    public Boolean decodeResult(@Nullable Object obj) {
      return (obj instanceof Boolean) && (boolean) obj;
    }
  }

  public void onCreateWindow(CreateWindowAction createWindowAction, @NonNull CreateWindowCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) {
      callback.defaultBehaviour(null);
      return;
    }
    channel.invokeMethod("onCreateWindow", createWindowAction.toMap(), callback);
  }

  public void onCloseWindow() {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    channel.invokeMethod("onCloseWindow", obj);
  }

  public static class GeolocationPermissionsShowPromptCallback extends BaseCallbackResultImpl<GeolocationPermissionShowPromptResponse> {
    @Nullable
    @Override
    public GeolocationPermissionShowPromptResponse decodeResult(@Nullable Object obj) {
      return GeolocationPermissionShowPromptResponse.fromMap((Map<String, Object>) obj);
    }
  }

  public void onGeolocationPermissionsShowPrompt(String origin, @NonNull GeolocationPermissionsShowPromptCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) {
      callback.defaultBehaviour(null);
      return;
    }
    Map<String, Object> obj = new HashMap<>();
    obj.put("origin", origin);
    channel.invokeMethod("onGeolocationPermissionsShowPrompt", obj, callback);
  }

  public void onGeolocationPermissionsHidePrompt() {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    channel.invokeMethod("onGeolocationPermissionsHidePrompt", obj);
  }

  public void onConsoleMessage(String message, int messageLevel) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("message", message);
    obj.put("messageLevel", messageLevel);
    channel.invokeMethod("onConsoleMessage", obj);
  }

  public void onProgressChanged(int progress) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("progress", progress);
    channel.invokeMethod("onProgressChanged", obj);
  }

  public void onTitleChanged(String title) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("title", title);
    channel.invokeMethod("onTitleChanged", obj);
  }

  public void onReceivedIcon(byte[] icon) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("icon", icon);
    channel.invokeMethod("onReceivedIcon", obj);
  }

  public void onReceivedTouchIconUrl(String url, boolean precomposed) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("url", url);
    obj.put("precomposed", precomposed);
    channel.invokeMethod("onReceivedTouchIconUrl", obj);
  }

  public static class PermissionRequestCallback extends BaseCallbackResultImpl<PermissionResponse> {
    @Nullable
    @Override
    public PermissionResponse decodeResult(@Nullable Object obj) {
      return PermissionResponse.fromMap((Map<String, Object>) obj);
    }
  }

  public void onPermissionRequest(String origin, List<String> resources, Object frame, @NonNull PermissionRequestCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) {
      callback.defaultBehaviour(null);
      return;
    }
    Map<String, Object> obj = new HashMap<>();
    obj.put("origin", origin);
    obj.put("resources", resources);
    obj.put("frame", frame);
    channel.invokeMethod("onPermissionRequest", obj, callback);
  }

  public void onPermissionRequestCanceled(String origin, List<String> resources) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("origin", origin);
    obj.put("resources", resources);
    channel.invokeMethod("onPermissionRequestCanceled", obj);
  }

  public static class ShouldOverrideUrlLoadingCallback extends BaseCallbackResultImpl<NavigationActionPolicy> {
    @Nullable
    @Override
    public NavigationActionPolicy decodeResult(@Nullable Object obj) {
      Integer action = obj instanceof Integer ? (Integer) obj : NavigationActionPolicy.CANCEL.rawValue();
      return NavigationActionPolicy.fromValue(action);
    }
  }

  public void shouldOverrideUrlLoading(NavigationAction navigationAction, @NonNull ShouldOverrideUrlLoadingCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) {
      callback.defaultBehaviour(null);
      return;
    }
    channel.invokeMethod("shouldOverrideUrlLoading", navigationAction.toMap(), callback);
  }

  public void onLoadStart(String url) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("url", url);
    channel.invokeMethod("onLoadStart", obj);
  }

  public void onLoadStop(String url) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("url", url);
    channel.invokeMethod("onLoadStop", obj);
  }

  public void onUpdateVisitedHistory(String url, boolean isReload) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("url", url);
    obj.put("isReload", isReload);
    channel.invokeMethod("onUpdateVisitedHistory", obj);
  }

  public void onReceivedError(WebResourceRequestExt request, WebResourceErrorExt error) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("request", request.toMap());
    obj.put("error", error.toMap());
    channel.invokeMethod("onReceivedError", obj);
  }

  public void onReceivedHttpError(WebResourceRequestExt request, WebResourceResponseExt errorResponse) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("request", request.toMap());
    obj.put("errorResponse", errorResponse.toMap());
    channel.invokeMethod("onReceivedHttpError", obj);
  }

  public static class ReceivedHttpAuthRequestCallback extends BaseCallbackResultImpl<HttpAuthResponse> {
    @Nullable
    @Override
    public HttpAuthResponse decodeResult(@Nullable Object obj) {
      return HttpAuthResponse.fromMap((Map<String, Object>) obj);
    }
  }

  public void onReceivedHttpAuthRequest(HttpAuthenticationChallenge challenge, @NonNull ReceivedHttpAuthRequestCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) {
      callback.defaultBehaviour(null);
      return;
    }
    channel.invokeMethod("onReceivedHttpAuthRequest", challenge.toMap(), callback);
  }

  public static class ReceivedServerTrustAuthRequestCallback extends BaseCallbackResultImpl<ServerTrustAuthResponse> {
    @Nullable
    @Override
    public ServerTrustAuthResponse decodeResult(@Nullable Object obj) {
      return ServerTrustAuthResponse.fromMap((Map<String, Object>) obj);
    }
  }

  public void onReceivedServerTrustAuthRequest(ServerTrustChallenge challenge, @NonNull ReceivedServerTrustAuthRequestCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) {
      callback.defaultBehaviour(null);
      return;
    }
    channel.invokeMethod("onReceivedServerTrustAuthRequest", challenge.toMap(), callback);
  }

  public static class ReceivedClientCertRequestCallback extends BaseCallbackResultImpl<ClientCertResponse> {
    @Nullable
    @Override
    public ClientCertResponse decodeResult(@Nullable Object obj) {
      return ClientCertResponse.fromMap((Map<String, Object>) obj);
    }
  }

  public void onReceivedClientCertRequest(ClientCertChallenge challenge, @NonNull ReceivedClientCertRequestCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) {
      callback.defaultBehaviour(null);
      return;
    }
    channel.invokeMethod("onReceivedClientCertRequest", challenge.toMap(), callback);
  }

  public void onZoomScaleChanged(float oldScale, float newScale) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("oldScale", oldScale);
    obj.put("newScale", newScale);
    channel.invokeMethod("onZoomScaleChanged", obj);
  }

  public static class SafeBrowsingHitCallback extends BaseCallbackResultImpl<SafeBrowsingResponse> {
    @Nullable
    @Override
    public SafeBrowsingResponse decodeResult(@Nullable Object obj) {
      return SafeBrowsingResponse.fromMap((Map<String, Object>) obj);
    }
  }

  public void onSafeBrowsingHit(String url, int threatType, @NonNull SafeBrowsingHitCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) {
      callback.defaultBehaviour(null);
      return;
    }
    Map<String, Object> obj = new HashMap<>();
    obj.put("url", url);
    obj.put("threatType", threatType);
    channel.invokeMethod("onSafeBrowsingHit", obj, callback);
  }

  public static class FormResubmissionCallback extends BaseCallbackResultImpl<Integer> {
    @Nullable
    @Override
    public Integer decodeResult(@Nullable Object obj) {
      return obj instanceof Integer ? (Integer) obj : null;
    }
  }

  public void onFormResubmission(String url, @NonNull FormResubmissionCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) {
      callback.defaultBehaviour(null);
      return;
    }
    Map<String, Object> obj = new HashMap<>();
    obj.put("url", url);
    channel.invokeMethod("onFormResubmission", obj, callback);
  }

  public void onPageCommitVisible(String url) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("url", url);
    channel.invokeMethod("onPageCommitVisible", obj);
  }

  public void onRenderProcessGone(boolean didCrash, int rendererPriorityAtExit) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("didCrash", didCrash);
    obj.put("rendererPriorityAtExit", rendererPriorityAtExit);
    channel.invokeMethod("onRenderProcessGone", obj);
  }

  public void onReceivedLoginRequest(String realm, String account, String args) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("realm", realm);
    obj.put("account", account);
    obj.put("args", args);
    channel.invokeMethod("onReceivedLoginRequest", obj);
  }

  public static class LoadResourceWithCustomSchemeCallback extends BaseCallbackResultImpl<CustomSchemeResponse> {
    @Nullable
    @Override
    public CustomSchemeResponse decodeResult(@Nullable Object obj) {
      return CustomSchemeResponse.fromMap((Map<String, Object>) obj);
    }
  }

  public void onLoadResourceWithCustomScheme(WebResourceRequestExt request, @NonNull LoadResourceWithCustomSchemeCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) {
      callback.defaultBehaviour(null);
      return;
    }
    Map<String, Object> obj = new HashMap<>();
    obj.put("request", request.toMap());
    channel.invokeMethod("onLoadResourceWithCustomScheme", obj, callback);
  }

  public static class SyncLoadResourceWithCustomSchemeCallback extends SyncBaseCallbackResultImpl<CustomSchemeResponse> {
    @Nullable
    @Override
    public CustomSchemeResponse decodeResult(@Nullable Object obj) {
      return (new LoadResourceWithCustomSchemeCallback()).decodeResult(obj);
    }
  }

  @Nullable
  public CustomSchemeResponse onLoadResourceWithCustomScheme(WebResourceRequestExt request) throws InterruptedException {
    MethodChannel channel = getChannel();
    if (channel == null) return null;
    final Map<String, Object> obj = new HashMap<>();
    obj.put("request", request.toMap());
    final SyncLoadResourceWithCustomSchemeCallback callback = new SyncLoadResourceWithCustomSchemeCallback();
    return Util.invokeMethodAndWaitResult(channel, "onLoadResourceWithCustomScheme", obj, callback);
  }

  public static class ShouldInterceptRequestCallback extends BaseCallbackResultImpl<WebResourceResponseExt> {
    @Nullable
    @Override
    public WebResourceResponseExt decodeResult(@Nullable Object obj) {
      return WebResourceResponseExt.fromMap((Map<String, Object>) obj);
    }
  }

  public void shouldInterceptRequest(WebResourceRequestExt request, @NonNull ShouldInterceptRequestCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) {
      callback.defaultBehaviour(null);
      return;
    }
    channel.invokeMethod("shouldInterceptRequest", request.toMap(), callback);
  }

  public static class SyncShouldInterceptRequestCallback extends SyncBaseCallbackResultImpl<WebResourceResponseExt> {
    @Nullable
    @Override
    public WebResourceResponseExt decodeResult(@Nullable Object obj) {
      return (new ShouldInterceptRequestCallback()).decodeResult(obj);
    }
  }

  @Nullable
  public WebResourceResponseExt shouldInterceptRequest(WebResourceRequestExt request) throws InterruptedException {
    MethodChannel channel = getChannel();
    if (channel == null) return null;
    final SyncShouldInterceptRequestCallback callback = new SyncShouldInterceptRequestCallback();
    return Util.invokeMethodAndWaitResult(channel, "shouldInterceptRequest", request.toMap(), callback);
  }

  public static class RenderProcessUnresponsiveCallback extends BaseCallbackResultImpl<Integer> {
    @Nullable
    @Override
    public Integer decodeResult(@Nullable Object obj) {
      return obj instanceof Integer ? (Integer) obj : null;
    }
  }

  public void onRenderProcessUnresponsive(String url, @NonNull RenderProcessUnresponsiveCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) {
      callback.defaultBehaviour(null);
      return;
    }
    Map<String, Object> obj = new HashMap<>();
    obj.put("url", url);
    channel.invokeMethod("onRenderProcessUnresponsive", obj, callback);
  }

  public static class RenderProcessResponsiveCallback extends BaseCallbackResultImpl<Integer> {
    @Nullable
    @Override
    public Integer decodeResult(@Nullable Object obj) {
      return obj instanceof Integer ? (Integer) obj : null;
    }
  }

  public void onRenderProcessResponsive(String url, @NonNull RenderProcessResponsiveCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) {
      callback.defaultBehaviour(null);
      return;
    }
    Map<String, Object> obj = new HashMap<>();
    obj.put("url", url);
    channel.invokeMethod("onRenderProcessResponsive", obj, callback);
  }

  public static class CallJsHandlerCallback extends BaseCallbackResultImpl<Object> {
    @Nullable
    @Override
    public Object decodeResult(@Nullable Object obj) {
      return obj;
    }
  }

  public void onCallJsHandler(String handlerName, String args, @NonNull CallJsHandlerCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) {
      callback.defaultBehaviour(null);
      return;
    }
    Map<String, Object> obj = new HashMap<>();
    obj.put("handlerName", handlerName);
    obj.put("args", args);
    channel.invokeMethod("onCallJsHandler", obj, callback);
  }

  public static class PrintRequestCallback extends BaseCallbackResultImpl<Boolean> {
    @Nullable
    @Override
    public Boolean decodeResult(@Nullable Object obj) {
      return (obj instanceof Boolean) && (boolean) obj;
    }
  }

  public void onPrintRequest(String url, String printJobId, @NonNull PrintRequestCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) {
      callback.defaultBehaviour(null);
      return;
    }
    Map<String, Object> obj = new HashMap<>();
    obj.put("url", url);
    obj.put("printJobId", printJobId);
    channel.invokeMethod("onPrintRequest", obj, callback);
  }

  public void onRequestFocus() {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    channel.invokeMethod("onRequestFocus", obj);
  }

  @Override
  public void dispose() {
    super.dispose();
    webView = null;
  }
}
