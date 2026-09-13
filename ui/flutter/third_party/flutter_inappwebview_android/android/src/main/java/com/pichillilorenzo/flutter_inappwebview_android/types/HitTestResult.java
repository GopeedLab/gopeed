package com.pichillilorenzo.flutter_inappwebview_android.types;

import android.webkit.WebView;

import androidx.annotation.Nullable;

import java.util.HashMap;
import java.util.Map;

public class HitTestResult {
  private int type;
  @Nullable
  private String extra;

  public HitTestResult(int type, @Nullable String extra) {
    this.type = type;
    this.extra = extra;
  }

  @Nullable
  static public HitTestResult fromWebViewHitTestResult(@Nullable WebView.HitTestResult hitTestResult) {
    if (hitTestResult == null) {
      return null;
    }
    
    return new HitTestResult(hitTestResult.getType(), hitTestResult.getExtra());
  }

  public int getType() {
    return type;
  }

  public void setType(int type) {
    this.type = type;
  }

  @Nullable
  public String getExtra() {
    return extra;
  }

  public void setExtra(@Nullable String extra) {
    this.extra = extra;
  }

  @Nullable
  public Map<String, Object> toMap() {
    Map<String, Object> hitTestResultMap = new HashMap<>();
    hitTestResultMap.put("type", type);
    hitTestResultMap.put("extra", extra);
    return hitTestResultMap;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    HitTestResult that = (HitTestResult) o;

    if (type != that.type) return false;
    return extra != null ? extra.equals(that.extra) : that.extra == null;
  }

  @Override
  public int hashCode() {
    int result = type;
    result = 31 * result + (extra != null ? extra.hashCode() : 0);
    return result;
  }

  @Override
  public String toString() {
    return "HitTestResultMap{" +
            "type=" + type +
            ", extra='" + extra + '\'' +
            '}';
  }
}
