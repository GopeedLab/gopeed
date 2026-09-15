package com.pichillilorenzo.flutter_inappwebview_android;

import android.os.Build;
import android.util.Log;
import android.webkit.CookieManager;
import android.webkit.CookieSyncManager;
import android.webkit.ValueCallback;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.webkit.CookieManagerCompat;
import androidx.webkit.WebViewFeature;

import com.pichillilorenzo.flutter_inappwebview_android.types.ChannelDelegateImpl;

import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Date;
import java.util.HashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.TimeZone;

import io.flutter.plugin.common.MethodCall;
import io.flutter.plugin.common.MethodChannel;

public class MyCookieManager extends ChannelDelegateImpl {
  protected static final String LOG_TAG = "MyCookieManager";
  public static final String METHOD_CHANNEL_NAME = "com.pichillilorenzo/flutter_inappwebview_cookiemanager";
  @Nullable
  public static CookieManager cookieManager;
  private CookieManager selectedCookieManager;
  @Nullable
  public InAppWebViewFlutterPlugin plugin;

  public MyCookieManager(@NonNull final InAppWebViewFlutterPlugin plugin) {
    super(new MethodChannel(plugin.messenger, METHOD_CHANNEL_NAME));
    this.plugin = plugin;
  }

  public static void init() {
    if (cookieManager == null) {
      cookieManager = getCookieManager();
    }
  }

  @Override
  public void onMethodCall(@NonNull MethodCall call, @NonNull MethodChannel.Result result) {
    if (call.method.equals("gopeed.removeProfile")) {
      GopeedProfiles.remove(call, result);
      return;
    }
    if (call.method.equals("gopeed.prepareProfile")) {
      GopeedProfiles.prepare(call, result);
      return;
    }
    String profileID = call.argument("gopeedProfileId");
    if (profileID != null && !profileID.isEmpty()) {
      if (!androidx.webkit.WebViewFeature.isFeatureSupported(androidx.webkit.WebViewFeature.MULTI_PROFILE) ||
          androidx.webkit.ProfileStore.getInstance().getProfile(profileID) == null) {
        result.error("UNAVAILABLE", "WebView profile is not initialized", null);
        return;
      }
      selectedCookieManager = androidx.webkit.ProfileStore.getInstance().getProfile(profileID).getCookieManager();
    }
    init();
    try {
    switch (call.method) {
      case "setCookie":
        {
          String url = (String) call.argument("url");
          String name = (String) call.argument("name");
          String value = (String) call.argument("value");
          String domain = (String) call.argument("domain");
          String path = (String) call.argument("path");
          String expiresDateString = (String) call.argument("expiresDate");
          Long expiresDate = (expiresDateString != null ? new Long(expiresDateString) : null);
          Integer maxAge = (Integer) call.argument("maxAge");
          Boolean isSecure = (Boolean) call.argument("isSecure");
          Boolean isHttpOnly = (Boolean) call.argument("isHttpOnly");
          String sameSite = (String) call.argument("sameSite");
          setCookie(url,
                  name,
                  value,
                  domain,
                  path,
                  expiresDate,
                  maxAge,
                  isSecure,
                  isHttpOnly,
                  sameSite,
                  result);
        }
        break;
      case "getCookies":
        result.success(getCookies((String) call.argument("url")));
        break;
      case "deleteCookie":
        {
          String url = (String) call.argument("url");
          String name = (String) call.argument("name");
          String domain = (String) call.argument("domain");
          String path = (String) call.argument("path");
          deleteCookie(url, name, domain, path, result);
        }
        break;
      case "deleteCookies":
        {
          String url = (String) call.argument("url");
          String domain = (String) call.argument("domain");
          String path = (String) call.argument("path");
          deleteCookies(url, domain, path, result);
        }
        break;
      case "deleteAllCookies":
        deleteAllCookies(result);
        break;
      case "removeSessionCookies":
        removeSessionCookies(result);
        break;
      default:
        result.notImplemented();
    }
    } finally { selectedCookieManager = null; }
  }

  /**
   * Instantiating CookieManager will load the Chromium task taking a 100ish ms so we do it lazily
   * to make sure it's done on a background thread as needed.
   *
   * https://github.com/facebook/react-native/blob/1903f6680d9750e244d97c3cd4a9f755a9a47c61/ReactAndroid/src/main/java/com/facebook/react/modules/network/ForwardingCookieHandler.java#L132
   */
  static private @Nullable CookieManager getCookieManager() {
    if (cookieManager == null) {
      try {
        cookieManager = CookieManager.getInstance();
      } catch (IllegalArgumentException ex) {
        // https://bugs.chromium.org/p/chromium/issues/detail?id=559720
        return null;
      } catch (Exception exception) {
        String message = exception.getMessage();
        // We cannot catch MissingWebViewPackageException as it is in a private / system API
        // class. This validates the exception's message to ensure we are only handling this
        // specific exception.
        // https://android.googlesource.com/platform/frameworks/base/+/master/core/java/android/webkit/WebViewFactory.java#348
        if (message != null
                && exception
                .getClass()
                .getCanonicalName()
                .equals("android.webkit.WebViewFactory.MissingWebViewPackageException")) {
          return null;
        } else {
          throw exception;
        }
      }
    }

    return cookieManager;
  }

  public void setCookie(String url,
                               String name,
                               String value,
                               String domain,
                               String path,
                               Long expiresDate,
                               Integer maxAge,
                               Boolean isSecure,
                               Boolean isHttpOnly,
                               String sameSite,
                               final MethodChannel.Result result) {
    final CookieManager cookieManager = selectedCookieManager != null ? selectedCookieManager : getCookieManager();
    if (cookieManager == null) {
      result.success(false);
      return;
    }

    String cookieValue = name + "=" + value + "; Path=" + path;

    if (domain != null)
      cookieValue += "; Domain=" + domain;

    if (expiresDate != null)
      cookieValue += "; Expires=" + getCookieExpirationDate(expiresDate);

    if (maxAge != null)
      cookieValue += "; Max-Age=" + maxAge.toString();

    if (isSecure != null && isSecure)
      cookieValue += "; Secure";

    if (isHttpOnly != null && isHttpOnly)
      cookieValue += "; HttpOnly";

    if (sameSite != null)
      cookieValue += "; SameSite=" + sameSite;

    cookieValue += ";";

    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
      cookieManager.setCookie(url, cookieValue, new ValueCallback<Boolean>() {
        @Override
        public void onReceiveValue(Boolean successful) {
          result.success(successful);
        }
      });
      cookieManager.flush();
    }
    else if (plugin != null) {
      CookieSyncManager cookieSyncMngr = CookieSyncManager.createInstance(plugin.applicationContext);
      cookieSyncMngr.startSync();
      cookieManager.setCookie(url, cookieValue);
      cookieSyncMngr.stopSync();
      cookieSyncMngr.sync();
      result.success(true);
    } else {
      cookieManager.setCookie(url, cookieValue);
      result.success(true);
    }
  }

  public List<Map<String, Object>> getCookies(final String url) {

    final List<Map<String, Object>> cookieListMap = new ArrayList<>();

    final CookieManager cookieManager = selectedCookieManager != null ? selectedCookieManager : getCookieManager();
    if (cookieManager == null) return cookieListMap;

    List<String> cookies = new ArrayList<>();
    if (WebViewFeature.isFeatureSupported(WebViewFeature.GET_COOKIE_INFO)) {
      cookies = CookieManagerCompat.getCookieInfo(cookieManager, url);
    } else {
      String cookiesString = cookieManager.getCookie(url);
      if (cookiesString != null) {
        cookies = Arrays.asList(cookiesString.split(";"));
      }
    }

    for (String cookie : cookies) {
      String[] cookieParams = cookie.split(";");
      if (cookieParams.length == 0) continue;

      String[] nameValue = cookieParams[0].split("=", 2);
      String name = nameValue[0].trim();
      String value = (nameValue.length > 1) ? nameValue[1].trim() : "";

      Map<String, Object> cookieMap = new HashMap<>();
      cookieMap.put("name", name);
      cookieMap.put("value", value);
      cookieMap.put("expiresDate", null);
      cookieMap.put("isSessionOnly", null);
      cookieMap.put("domain", null);
      cookieMap.put("sameSite", null);
      cookieMap.put("isSecure", null);
      cookieMap.put("isHttpOnly", null);
      cookieMap.put("path", null);

      if (WebViewFeature.isFeatureSupported(WebViewFeature.GET_COOKIE_INFO)) {
        cookieMap.put("isSecure", false);
        cookieMap.put("isHttpOnly", false);

        for (int i = 1; i < cookieParams.length; i++) {
          String[] cookieParamNameValue = cookieParams[i].split("=", 2);
          String cookieParamName = cookieParamNameValue[0].trim();
          String cookieParamValue = (cookieParamNameValue.length > 1) ? cookieParamNameValue[1].trim() : "";

          if (cookieParamName.equalsIgnoreCase("Expires")) {
            try {
              final SimpleDateFormat sdf = new SimpleDateFormat("EEE, dd MMM yyyy hh:mm:ss z", Locale.US);
              Date expiryDate = sdf.parse(cookieParamValue);
              if (expiryDate != null) {
                cookieMap.put("expiresDate", expiryDate.getTime());
              }
            } catch (ParseException e) {
              Log.e(LOG_TAG, "", e);
            }
          } else if (cookieParamName.equalsIgnoreCase("Max-Age")) {
            try {
              long maxAge = Long.parseLong(cookieParamValue);
              cookieMap.put("expiresDate", System.currentTimeMillis() + maxAge);
            } catch (NumberFormatException e) {
              Log.e(LOG_TAG, "", e);
            }
          } else if (cookieParamName.equalsIgnoreCase("Domain")) {
            cookieMap.put("domain", cookieParamValue);
          } else if (cookieParamName.equalsIgnoreCase("SameSite")) {
            cookieMap.put("sameSite", cookieParamValue);
          } else if (cookieParamName.equalsIgnoreCase("Secure")) {
            cookieMap.put("isSecure", true);
          } else if (cookieParamName.equalsIgnoreCase("HttpOnly")) {
            cookieMap.put("isHttpOnly", true);
          } else if (cookieParamName.equalsIgnoreCase("Path")) {
            cookieMap.put("path", cookieParamValue);
          }
        }
      }

      cookieListMap.add(cookieMap);
    }
    return cookieListMap;

  }

  public void deleteCookie(String url, String name, String domain, String path, final MethodChannel.Result result) {
    final CookieManager cookieManager = selectedCookieManager != null ? selectedCookieManager : getCookieManager();
    if (cookieManager == null) {
      result.success(false);
      return;
    }

    String cookieValue = name + "=; Path=" + path + "; Max-Age=-1";

    if (domain != null)
      cookieValue += "; Domain=" + domain;

    cookieValue += ";";

    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
      cookieManager.setCookie(url, cookieValue, new ValueCallback<Boolean>() {
        @Override
        public void onReceiveValue(Boolean successful) {
          result.success(successful);
        }
      });
      cookieManager.flush();
    }
    else if (plugin != null) {
      CookieSyncManager cookieSyncMngr = CookieSyncManager.createInstance(plugin.applicationContext);
      cookieSyncMngr.startSync();
      cookieManager.setCookie(url, cookieValue);
      cookieSyncMngr.stopSync();
      cookieSyncMngr.sync();
      result.success(true);
    } else {
      cookieManager.setCookie(url, cookieValue);
      result.success(true);
    }
  }

  public void deleteCookies(String url, String domain, String path, final MethodChannel.Result result) {
    final CookieManager cookieManager = selectedCookieManager != null ? selectedCookieManager : getCookieManager();
    if (cookieManager == null) {
      result.success(false);
      return;
    }

    CookieSyncManager cookieSyncMngr = null;

    String cookiesString = cookieManager.getCookie(url);
    if (cookiesString != null) {

      if (Build.VERSION.SDK_INT < Build.VERSION_CODES.LOLLIPOP && plugin != null) {
        cookieSyncMngr = CookieSyncManager.createInstance(plugin.applicationContext);
        cookieSyncMngr.startSync();
      }

      String[] cookies = cookiesString.split(";");
      for (String cookie : cookies) {
        String[] nameValue = cookie.split("=", 2);
        String name = nameValue[0].trim();

        String cookieValue = name + "=; Path=" + path + "; Max-Age=-1";

        if (domain != null)
          cookieValue += "; Domain=" + domain;

        cookieValue += ";";

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP)
          cookieManager.setCookie(url, cookieValue, null);
        else
          cookieManager.setCookie(url, cookieValue);
      }

      if (cookieSyncMngr != null) {
        cookieSyncMngr.stopSync();
        cookieSyncMngr.sync();
      } else if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP)
        cookieManager.flush();
    }
    result.success(true);
  }

  public void deleteAllCookies(final MethodChannel.Result result) {
    final CookieManager cookieManager = selectedCookieManager != null ? selectedCookieManager : getCookieManager();
    if (cookieManager == null) {
      result.success(false);
      return;
    }

    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
      cookieManager.removeAllCookies(new ValueCallback<Boolean>() {
        @Override
        public void onReceiveValue(Boolean successful) {
          result.success(successful);
        }
      });
      cookieManager.flush();
    }
    else if (plugin != null) {
      CookieSyncManager cookieSyncMngr = CookieSyncManager.createInstance(plugin.applicationContext);
      cookieSyncMngr.startSync();
      cookieManager.removeAllCookie();
      cookieSyncMngr.stopSync();
      cookieSyncMngr.sync();
      result.success(true);
    } else {
      cookieManager.removeAllCookie();
      result.success(true);
    }
  }

  public void removeSessionCookies(final MethodChannel.Result result) {
    final CookieManager cookieManager = selectedCookieManager != null ? selectedCookieManager : getCookieManager();
    if (cookieManager == null) {
      result.success(false);
      return;
    }

    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
      cookieManager.removeSessionCookies(new ValueCallback<Boolean>() {
        @Override
        public void onReceiveValue(Boolean successful) {
          result.success(successful);
        }
      });
      cookieManager.flush();
    }
    else if (plugin != null) {
      CookieSyncManager cookieSyncMngr = CookieSyncManager.createInstance(plugin.applicationContext);
      cookieSyncMngr.startSync();
      cookieManager.removeSessionCookie();
      cookieSyncMngr.stopSync();
      cookieSyncMngr.sync();
      result.success(true);
    } else {
      cookieManager.removeSessionCookie();
      result.success(true);
    }
  }

  public static String getCookieExpirationDate(Long timestamp) {
    final SimpleDateFormat sdf = new SimpleDateFormat("EEE, dd MMM yyyy hh:mm:ss z", Locale.US);
    sdf.setTimeZone(TimeZone.getTimeZone("GMT"));
    return sdf.format(new Date(timestamp));
  }

  @Override
  public void dispose() {
    super.dispose();
    plugin = null;
  }
}
