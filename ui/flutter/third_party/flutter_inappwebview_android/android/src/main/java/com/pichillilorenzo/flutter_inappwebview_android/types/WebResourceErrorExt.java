package com.pichillilorenzo.flutter_inappwebview_android.types;

import android.os.Build;
import android.webkit.WebResourceError;

import androidx.annotation.NonNull;
import androidx.annotation.RequiresApi;
import androidx.webkit.WebResourceErrorCompat;
import androidx.webkit.WebViewFeature;

import java.util.HashMap;
import java.util.Map;

public class WebResourceErrorExt {
  private int type;
  @NonNull
  private String description;

  public WebResourceErrorExt(int type, @NonNull String description) {
    this.type = type;
    this.description = description;
  }

  @RequiresApi(Build.VERSION_CODES.M)
  static public WebResourceErrorExt fromWebResourceError(@NonNull WebResourceError error) {
    return new WebResourceErrorExt(error.getErrorCode(), error.getDescription().toString());
  }

  @RequiresApi(Build.VERSION_CODES.M)
  static public WebResourceErrorExt fromWebResourceError(@NonNull WebResourceErrorCompat error) {
    int type = -1;
    if (WebViewFeature.isFeatureSupported(WebViewFeature.WEB_RESOURCE_ERROR_GET_CODE)) {
      type = error.getErrorCode();
    }
    String description = "";
    if (WebViewFeature.isFeatureSupported(WebViewFeature.WEB_RESOURCE_ERROR_GET_DESCRIPTION)) {
      description = error.getDescription().toString();
    }
    return new WebResourceErrorExt(type, description);
  }

  public Map<String, Object> toMap() {
    Map<String, Object> webResourceErrorMap = new HashMap<>();
    webResourceErrorMap.put("type", getType());
    webResourceErrorMap.put("description", getDescription());
    return webResourceErrorMap;
  }

  public int getType() {
    return type;
  }

  public void setType(int type) {
    this.type = type;
  }

  @NonNull
  public String getDescription() {
    return description;
  }

  public void setDescription(@NonNull String description) {
    this.description = description;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    WebResourceErrorExt that = (WebResourceErrorExt) o;

    if (type != that.type) return false;
    return description.equals(that.description);
  }

  @Override
  public int hashCode() {
    int result = type;
    result = 31 * result + description.hashCode();
    return result;
  }

  @Override
  public String toString() {
    return "WebResourceErrorExt{" +
            "type=" + type +
            ", description='" + description + '\'' +
            '}';
  }
}
