package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import java.util.HashMap;
import java.util.Map;

public class WebMessagePortCompatExt {
  private int index;
  @NonNull
  private String webMessageChannelId;

  public WebMessagePortCompatExt(int index, @NonNull String webMessageChannelId) {
    this.index = index;
    this.webMessageChannelId = webMessageChannelId;
  }

  @Nullable
  public static WebMessagePortCompatExt fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    Integer index = (Integer) map.get("index");
    String webMessageChannelId = (String) map.get("webMessageChannelId");
    return new WebMessagePortCompatExt(index, webMessageChannelId);
  }

  public Map<String, Object> toMap() {
    Map<String, Object> proxyRuleMap = new HashMap<>();
    proxyRuleMap.put("index", index);
    proxyRuleMap.put("webMessageChannelId", webMessageChannelId);
    return proxyRuleMap;
  }

  public int getIndex() {
    return index;
  }

  public void setIndex(int index) {
    this.index = index;
  }

  @NonNull
  public String getWebMessageChannelId() {
    return webMessageChannelId;
  }

  public void setWebMessageChannelId(@NonNull String webMessageChannelId) {
    this.webMessageChannelId = webMessageChannelId;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    WebMessagePortCompatExt that = (WebMessagePortCompatExt) o;

    if (index != that.index) return false;
    return webMessageChannelId.equals(that.webMessageChannelId);
  }

  @Override
  public int hashCode() {
    int result = index;
    result = 31 * result + webMessageChannelId.hashCode();
    return result;
  }

  @Override
  public String toString() {
    return "WebMessagePortCompatExt{" +
            "index=" + index +
            ", webMessageChannelId='" + webMessageChannelId + '\'' +
            '}';
  }
}
