package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.webkit.WebMessageCompat;
import androidx.webkit.WebViewFeature;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;

public class WebMessageCompatExt {
  @Nullable
  private Object data;
  private @WebMessageCompat.Type int type;

  @Nullable
  private List<WebMessagePortCompatExt> ports;

  public WebMessageCompatExt(@Nullable Object data, int type, @Nullable List<WebMessagePortCompatExt> ports) {
    this.data = data;
    this.type = type;
    this.ports = ports;
  }

  public static WebMessageCompatExt fromMapWebMessageCompat(@NonNull WebMessageCompat message) {
    Object data;
    if (WebViewFeature.isFeatureSupported(WebViewFeature.WEB_MESSAGE_ARRAY_BUFFER) && message.getType() == WebMessageCompat.TYPE_ARRAY_BUFFER) {
      data = message.getArrayBuffer();
    } else {
      data = message.getData();
    }
    return new WebMessageCompatExt(data, message.getType(), null);
  }

  @Nullable
  public static WebMessageCompatExt fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    Object data = map.get("data");
    Integer type = (Integer) map.get("type");
    List<Map<String, Object>> portMapList = (List<Map<String, Object>>) map.get("ports");
    List<WebMessagePortCompatExt> ports = null;
    if (portMapList != null && !portMapList.isEmpty()) {
      ports = new ArrayList<>();
      for (Map<String, Object> portMap : portMapList) {
        ports.add(WebMessagePortCompatExt.fromMap(portMap));
      }
    }
    return new WebMessageCompatExt(data, type, ports);
  }

  public Map<String, Object> toMap() {
    Map<String, Object> proxyRuleMap = new HashMap<>();
    proxyRuleMap.put("data", data);
    proxyRuleMap.put("type", type);
    return proxyRuleMap;
  }

  @Nullable
  public Object getData() {
    return data;
  }

  public void setData(@Nullable Object data) {
    this.data = data;
  }

  public int getType() {
    return type;
  }

  public void setType(int type) {
    this.type = type;
  }

  @Nullable
  public List<WebMessagePortCompatExt> getPorts() {
    return ports;
  }

  public void setPorts(@Nullable List<WebMessagePortCompatExt> ports) {
    this.ports = ports;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    WebMessageCompatExt that = (WebMessageCompatExt) o;

    if (type != that.type) return false;
    if (!Objects.equals(data, that.data)) return false;
    return Objects.equals(ports, that.ports);
  }

  @Override
  public int hashCode() {
    int result = data != null ? data.hashCode() : 0;
    result = 31 * result + type;
    result = 31 * result + (ports != null ? ports.hashCode() : 0);
    return result;
  }

  @Override
  public String toString() {
    return "WebMessageCompatExt{" +
            "data=" + data +
            ", type=" + type +
            ", ports=" + ports +
            '}';
  }
}
