package com.pichillilorenzo.flutter_inappwebview_android.types;

import java.util.HashMap;
import java.util.Map;

public class NavigationAction {
  URLRequest request;
  boolean isForMainFrame;
  boolean hasGesture;
  boolean isRedirect;

  public NavigationAction(URLRequest request, boolean isForMainFrame, boolean hasGesture, boolean isRedirect) {
    this.request = request;
    this.isForMainFrame = isForMainFrame;
    this.hasGesture = hasGesture;
    this.isRedirect = isRedirect;
  }

  public Map<String, Object> toMap() {
    Map<String, Object> navigationActionMap = new HashMap<>();
    navigationActionMap.put("request", request.toMap());
    navigationActionMap.put("isForMainFrame", isForMainFrame);
    navigationActionMap.put("hasGesture", hasGesture);
    navigationActionMap.put("isRedirect", isRedirect);
    navigationActionMap.put("navigationType", null);
    navigationActionMap.put("sourceFrame", null);
    navigationActionMap.put("targetFrame", null);
    navigationActionMap.put("shouldPerformDownload", null);
    return navigationActionMap;
  }

  public URLRequest getRequest() {
    return request;
  }

  public void setRequest(URLRequest request) {
    this.request = request;
  }

  public boolean isForMainFrame() {
    return isForMainFrame;
  }

  public void setForMainFrame(boolean forMainFrame) {
    isForMainFrame = forMainFrame;
  }

  public boolean isHasGesture() {
    return hasGesture;
  }

  public void setHasGesture(boolean hasGesture) {
    this.hasGesture = hasGesture;
  }

  public boolean isRedirect() {
    return isRedirect;
  }

  public void setRedirect(boolean redirect) {
    isRedirect = redirect;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    NavigationAction that = (NavigationAction) o;

    if (isForMainFrame != that.isForMainFrame) return false;
    if (hasGesture != that.hasGesture) return false;
    if (isRedirect != that.isRedirect) return false;
    return request.equals(that.request);
  }

  @Override
  public int hashCode() {
    int result = request.hashCode();
    result = 31 * result + (isForMainFrame ? 1 : 0);
    result = 31 * result + (hasGesture ? 1 : 0);
    result = 31 * result + (isRedirect ? 1 : 0);
    return result;
  }

  @Override
  public String toString() {
    return "NavigationAction{" +
            "request=" + request +
            ", isForMainFrame=" + isForMainFrame +
            ", hasGesture=" + hasGesture +
            ", isRedirect=" + isRedirect +
            '}';
  }
}
