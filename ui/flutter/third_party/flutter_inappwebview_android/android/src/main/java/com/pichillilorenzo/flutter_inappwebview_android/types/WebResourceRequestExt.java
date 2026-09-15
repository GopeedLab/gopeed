package com.pichillilorenzo.flutter_inappwebview_android.types;

import android.os.Build;
import android.webkit.WebResourceRequest;

import androidx.annotation.NonNull;
import androidx.annotation.RequiresApi;
import androidx.webkit.WebResourceRequestCompat;
import androidx.webkit.WebViewFeature;

import java.util.HashMap;
import java.util.Map;

public class WebResourceRequestExt {
  @NonNull
  private String url;
  private Map<String, String> headers;
  private boolean isRedirect;
  private boolean hasGesture;
  private boolean isForMainFrame;
  private String method;

  public WebResourceRequestExt(@NonNull String url, Map<String, String> headers, boolean isRedirect, boolean hasGesture, boolean isForMainFrame, String method) {
    this.url = url;
    this.headers = headers;
    this.isRedirect = isRedirect;
    this.hasGesture = hasGesture;
    this.isForMainFrame = isForMainFrame;
    this.method = method;
  }

  @RequiresApi(Build.VERSION_CODES.LOLLIPOP)
  static public WebResourceRequestExt fromWebResourceRequest(@NonNull WebResourceRequest request) { 
      boolean isRedirect = false;
      if (WebViewFeature.isFeatureSupported(WebViewFeature.WEB_RESOURCE_REQUEST_IS_REDIRECT)) {
        isRedirect = WebResourceRequestCompat.isRedirect(request);
      } else if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.N) {
        isRedirect = request.isRedirect();
      }
      return new WebResourceRequestExt(request.getUrl().toString(),
              request.getRequestHeaders(),
              isRedirect,
              request.hasGesture(),
              request.isForMainFrame(),
              request.getMethod()
      );
  }

  public Map<String, Object> toMap() {
    Map<String, Object> webResourceRequestMap = new HashMap<>();
    webResourceRequestMap.put("url", url);
    webResourceRequestMap.put("headers", headers);
    webResourceRequestMap.put("isRedirect", isRedirect);
    webResourceRequestMap.put("hasGesture", hasGesture);
    webResourceRequestMap.put("isForMainFrame", isForMainFrame);
    webResourceRequestMap.put("method", method);
    return webResourceRequestMap;
  }

  @NonNull
  public String getUrl() {
    return url;
  }

  public void setUrl(@NonNull String url) {
    this.url = url;
  }

  public Map<String, String> getHeaders() {
    return headers;
  }

  public void setHeaders(Map<String, String> headers) {
    this.headers = headers;
  }

  public boolean isRedirect() {
    return isRedirect;
  }

  public void setRedirect(boolean redirect) {
    isRedirect = redirect;
  }

  public boolean isHasGesture() {
    return hasGesture;
  }

  public void setHasGesture(boolean hasGesture) {
    this.hasGesture = hasGesture;
  }

  public boolean isForMainFrame() {
    return isForMainFrame;
  }

  public void setForMainFrame(boolean forMainFrame) {
    isForMainFrame = forMainFrame;
  }

  public String getMethod() {
    return method;
  }

  public void setMethod(String method) {
    this.method = method;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    WebResourceRequestExt that = (WebResourceRequestExt) o;

    if (isRedirect != that.isRedirect) return false;
    if (hasGesture != that.hasGesture) return false;
    if (isForMainFrame != that.isForMainFrame) return false;
    if (!url.equals(that.url)) return false;
    if (headers != null ? !headers.equals(that.headers) : that.headers != null) return false;
    return method != null ? method.equals(that.method) : that.method == null;
  }

  @Override
  public int hashCode() {
    int result = url.hashCode();
    result = 31 * result + (headers != null ? headers.hashCode() : 0);
    result = 31 * result + (isRedirect ? 1 : 0);
    result = 31 * result + (hasGesture ? 1 : 0);
    result = 31 * result + (isForMainFrame ? 1 : 0);
    result = 31 * result + (method != null ? method.hashCode() : 0);
    return result;
  }

  @Override
  public String toString() {
    return "WebResourceRequestExt{" +
            "url=" + url +
            ", headers=" + headers +
            ", isRedirect=" + isRedirect +
            ", hasGesture=" + hasGesture +
            ", isForMainFrame=" + isForMainFrame +
            ", method='" + method + '\'' +
            '}';
  }
}
