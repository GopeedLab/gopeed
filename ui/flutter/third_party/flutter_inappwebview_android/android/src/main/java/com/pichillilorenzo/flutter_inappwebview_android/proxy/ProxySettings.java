package com.pichillilorenzo.flutter_inappwebview_android.proxy;

import androidx.annotation.NonNull;
import androidx.webkit.ProxyConfig;

import com.pichillilorenzo.flutter_inappwebview_android.ISettings;
import com.pichillilorenzo.flutter_inappwebview_android.types.ProxyRuleExt;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class ProxySettings implements ISettings<ProxyConfig> {
  List<String> bypassRules = new ArrayList<>();
  List<String> directs = new ArrayList<>();
  List<ProxyRuleExt> proxyRules = new ArrayList<>();
  Boolean bypassSimpleHostnames = null;
  Boolean removeImplicitRules = null;
  Boolean reverseBypassEnabled = false;

  @NonNull
  @Override
  public ProxySettings parse(@NonNull Map<String, Object> settings) {
    for (Map.Entry<String, Object> pair : settings.entrySet()) {
      String key = pair.getKey();
      Object value = pair.getValue();
      if (value == null) {
        continue;
      }

      switch (key) {
        case "bypassRules":
          bypassRules = (List<String>) value;
          break;
        case "directs":
          directs = (List<String>) value;
          break;
        case "proxyRules":
          proxyRules = new ArrayList<>();
          List<Map<String, String>> proxyRuleMapList = (List<Map<String, String>>) value;
          for (Map<String, String> proxyRuleMap : proxyRuleMapList) {
            ProxyRuleExt proxyRuleExt = ProxyRuleExt.fromMap(proxyRuleMap);
            if (proxyRuleExt != null) {
              proxyRules.add(proxyRuleExt);
            }
          }
          break;
        case "bypassSimpleHostnames":
          bypassSimpleHostnames = (Boolean) value;
          break;
        case "removeImplicitRules":
          removeImplicitRules = (Boolean) value;
          break;
        case "reverseBypassEnabled":
          reverseBypassEnabled = (Boolean) value;
          break;
      }
    }

    return this;
  }

  @NonNull
  @Override
  public Map<String, Object> toMap() {
    List<Map<String, String>> proxyRuleMapList = new ArrayList<>();
    for (ProxyRuleExt proxyRuleExt : proxyRules) {
      proxyRuleMapList.add(proxyRuleExt.toMap());
    }
    Map<String, Object> settings = new HashMap<>();
    settings.put("bypassRules", bypassRules);
    settings.put("directs", directs);
    settings.put("proxyRules", proxyRuleMapList);
    settings.put("bypassSimpleHostnames", bypassSimpleHostnames);
    settings.put("removeImplicitRules", removeImplicitRules);
    settings.put("reverseBypassEnabled", reverseBypassEnabled);
    return settings;
  }

  @NonNull
  @Override
  public Map<String, Object> getRealSettings(@NonNull ProxyConfig proxyConfig) {
    Map<String, Object> realSettings = toMap();
    List<Map<String, String>> proxyRuleMapList = new ArrayList<>();
    List<ProxyConfig.ProxyRule> proxyRules = proxyConfig.getProxyRules();
    for (ProxyConfig.ProxyRule proxyRule : proxyRules) {
      Map<String, String> proxyRuleMap = new HashMap<>();
      proxyRuleMap.put("url", proxyRule.getUrl());
      proxyRuleMap.put("schemeFilter", proxyRule.getSchemeFilter());
      proxyRuleMapList.add(proxyRuleMap);
    }
    realSettings.put("bypassRules", proxyConfig.getBypassRules());
    realSettings.put("proxyRules", proxyRuleMapList);
    realSettings.put("reverseBypassEnabled", proxyConfig.isReverseBypassEnabled());
    return realSettings;
  }
}
