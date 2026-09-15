package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.webkit.ProxyConfig;

import java.util.HashMap;
import java.util.Map;

public class ProxyRuleExt {
  @Nullable
  @ProxyConfig.ProxyScheme
  private String schemeFilter;
  @NonNull
  private String url;

  public ProxyRuleExt(@Nullable @ProxyConfig.ProxyScheme String schemeFilter, @NonNull String url) {
    this.schemeFilter = schemeFilter;
    this.url = url;
  }

  @Nullable
  public static ProxyRuleExt fromMap(@Nullable Map<String, String> map) {
    if (map == null) {
      return null;
    }
    String url = (String) map.get("url");
    String schemeFilter = (String) map.get("schemeFilter");
    return new ProxyRuleExt(schemeFilter, url);
  }

  public Map<String, String> toMap() {
    Map<String, String> proxyRuleMap = new HashMap<>();
    proxyRuleMap.put("url", url);
    proxyRuleMap.put("schemeFilter", schemeFilter);
    return proxyRuleMap;
  }

  @Nullable
  public String getSchemeFilter() {
    return schemeFilter;
  }

  public void setSchemeFilter(@Nullable String schemeFilter) {
    this.schemeFilter = schemeFilter;
  }

  @NonNull
  public String getUrl() {
    return url;
  }

  public void setUrl(@NonNull String url) {
    this.url = url;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    ProxyRuleExt that = (ProxyRuleExt) o;

    if (schemeFilter != null ? !schemeFilter.equals(that.schemeFilter) : that.schemeFilter != null)
      return false;
    return url.equals(that.url);
  }

  @Override
  public int hashCode() {
    int result = schemeFilter != null ? schemeFilter.hashCode() : 0;
    result = 31 * result + url.hashCode();
    return result;
  }

  @Override
  public String toString() {
    return "ProxyRuleExt{" +
            "schemeFilter='" + schemeFilter + '\'' +
            ", url='" + url + '\'' +
            '}';
  }
}
