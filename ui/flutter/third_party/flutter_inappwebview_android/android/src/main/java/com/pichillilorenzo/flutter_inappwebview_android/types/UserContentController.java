package com.pichillilorenzo.flutter_inappwebview_android.types;

import android.annotation.SuppressLint;
import android.text.TextUtils;
import android.webkit.WebView;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.webkit.ScriptHandler;
import androidx.webkit.WebViewCompat;
import androidx.webkit.WebViewFeature;

import com.pichillilorenzo.flutter_inappwebview_android.Util;
import com.pichillilorenzo.flutter_inappwebview_android.plugin_scripts_js.JavaScriptBridgeJS;
import com.pichillilorenzo.flutter_inappwebview_android.plugin_scripts_js.PluginScriptsUtil;

import org.json.JSONObject;

import java.util.ArrayList;
import java.util.Collection;
import java.util.HashMap;
import java.util.HashSet;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

@SuppressLint("RestrictedApi")
public class UserContentController implements Disposable {
  protected static final String LOG_TAG = "UserContentController";

  @NonNull
  private final Set<ContentWorld> contentWorlds = new HashSet<ContentWorld>() {{
    add(ContentWorld.PAGE);
  }};

  private final Map<UserScript, ScriptHandler> scriptHandlerMap = new HashMap<>();

  @Nullable
  private ScriptHandler contentWorldsCreatorScript;

  @NonNull
  private final Map<UserScriptInjectionTime, LinkedHashSet<UserScript>> userOnlyScripts = new HashMap<UserScriptInjectionTime, LinkedHashSet<UserScript>>() {{
    put(UserScriptInjectionTime.AT_DOCUMENT_START, new LinkedHashSet<UserScript>());
    put(UserScriptInjectionTime.AT_DOCUMENT_END, new LinkedHashSet<UserScript>());
  }};
  @NonNull
  private final Map<UserScriptInjectionTime, LinkedHashSet<PluginScript>> pluginScripts = new HashMap<UserScriptInjectionTime, LinkedHashSet<PluginScript>>() {{
    put(UserScriptInjectionTime.AT_DOCUMENT_START, new LinkedHashSet<PluginScript>());
    put(UserScriptInjectionTime.AT_DOCUMENT_END, new LinkedHashSet<PluginScript>());
  }};

  @Nullable
  public WebView webView;

  public UserContentController(WebView webView) {
    this.webView = webView;
  }

  public String generateWrappedCodeForDocumentStart() {
    return Util.replaceAll(
            DOCUMENT_READY_WRAPPER_JS_SOURCE,
            PluginScriptsUtil.VAR_PLACEHOLDER_VALUE,
            generateCodeForDocumentStart());
  }

  public String generateWrappedCodeForDocumentEnd() {
    UserScriptInjectionTime injectionTime = UserScriptInjectionTime.AT_DOCUMENT_END;
    String js = "";
    if (!WebViewFeature.isFeatureSupported(WebViewFeature.DOCUMENT_START_SCRIPT)) {
      // try to reload scripts if they were not loaded during the AT_DOCUMENT_START event
      js += generateCodeForDocumentStart();
    }
    js += generatePluginScriptsCodeAt(injectionTime);
    js += generateUserOnlyScriptsCodeAt(injectionTime);
    js = USER_SCRIPTS_AT_DOCUMENT_END_WRAPPER_JS_SOURCE.replace(PluginScriptsUtil.VAR_PLACEHOLDER_VALUE, js);
    return js;
  }

  public String generateCodeForDocumentStart() {
    UserScriptInjectionTime injectionTime = UserScriptInjectionTime.AT_DOCUMENT_START;
    String js = "";
    js += generatePluginScriptsCodeAt(injectionTime);
    js += generateContentWorldsCreatorCode();
    js += generateUserOnlyScriptsCodeAt(injectionTime);
    js = USER_SCRIPTS_AT_DOCUMENT_START_WRAPPER_JS_SOURCE.replace(PluginScriptsUtil.VAR_PLACEHOLDER_VALUE, js);
    return js;
  }

  public String generateContentWorldsCreatorCode() {
    if (this.contentWorlds.size() == 1) {
      return "";
    }

    StringBuilder source = new StringBuilder();
    LinkedHashSet<PluginScript> pluginScriptsRequired = this.getPluginScriptsRequiredInAllContentWorlds();
    for (PluginScript script : pluginScriptsRequired) {
      source.append(script.getSource());
    }
    List<String> contentWorldsNames = new ArrayList<>();
    for (ContentWorld contentWorld : this.contentWorlds) {
      if (contentWorld.equals(ContentWorld.PAGE)) {
        continue;
      }
      contentWorldsNames.add("'" + escapeContentWorldName(contentWorld.getName()) + "'");
    }

    return CONTENT_WORLDS_GENERATOR_JS_SOURCE
            .replace(PluginScriptsUtil.VAR_CONTENT_WORLD_NAME_ARRAY, TextUtils.join(", ", contentWorldsNames))
            .replace(PluginScriptsUtil.VAR_JSON_SOURCE_ENCODED, escapeCode(source.toString()));

  }

  public String generatePluginScriptsCodeAt(UserScriptInjectionTime injectionTime) {
    StringBuilder js = new StringBuilder();
    LinkedHashSet<PluginScript> scripts = this.getPluginScriptsAt(injectionTime);
    for (PluginScript script : scripts) {
      String source = ";" + script.getSource();
      source = wrapSourceCodeInContentWorld(script.getContentWorld(), source);
      js.append(source);
    }
    return js.toString();
  }

  public String generateUserOnlyScriptsCodeAt(UserScriptInjectionTime injectionTime) {
    StringBuilder js = new StringBuilder();
    LinkedHashSet<UserScript> scripts = this.getUserOnlyScriptsAt(injectionTime);
    for (UserScript script : scripts) {
      String source = ";" + script.getSource();
      source = wrapSourceCodeInContentWorld(script.getContentWorld(), source);
      js.append(source);
    }
    return js.toString();
  }

  public String generateCodeForScriptEvaluation(String source, @Nullable ContentWorld contentWorld) {
    if (contentWorld != null && !contentWorld.equals(ContentWorld.PAGE)) {
      StringBuilder sourceWrapped = new StringBuilder();
      if (!contentWorlds.contains(contentWorld)) {
        contentWorlds.add(contentWorld);

        StringBuilder pluginScriptsSource = new StringBuilder();
        LinkedHashSet<PluginScript> pluginScriptsRequired = this.getPluginScriptsRequiredInAllContentWorlds();
        for (PluginScript script : pluginScriptsRequired) {
          pluginScriptsSource.append(script.getSource());
        }
        String contentWorldCreatorCode = CONTENT_WORLDS_GENERATOR_JS_SOURCE
                .replace(PluginScriptsUtil.VAR_CONTENT_WORLD_NAME_ARRAY, "'" + escapeContentWorldName(contentWorld.getName()) + "'")
                .replace(PluginScriptsUtil.VAR_JSON_SOURCE_ENCODED, escapeCode(pluginScriptsSource.toString()));
        sourceWrapped.append(contentWorldCreatorCode).append(";");
      }
      return sourceWrapped.append(wrapSourceCodeInContentWorld(contentWorld, source)).toString();
    }
    return source;
  }

  public String wrapSourceCodeInContentWorld(@Nullable ContentWorld contentWorld, String source) {
    String sourceWrapped = contentWorld == null || contentWorld.equals(ContentWorld.PAGE) ? source :
            CONTENT_WORLD_WRAPPER_JS_SOURCE
                    .replace(PluginScriptsUtil.VAR_CONTENT_WORLD_NAME, escapeContentWorldName(contentWorld.getName()))
                    .replace(PluginScriptsUtil.VAR_JSON_SOURCE_ENCODED, escapeCode(source));

    return sourceWrapped;
  }

  public static String escapeCode(String code) {
    String escapedCode = JSONObject.quote(code);
    // escapedCode = escapedCode.substring(1, escapedCode.length() - 1);
    return escapedCode;
  }

  public static String escapeContentWorldName(String name) {
    return name.replaceAll("'", "\\\\'");
  }

  public LinkedHashSet<UserScript> getUserOnlyScriptsAt(UserScriptInjectionTime injectionTime) {
    return new LinkedHashSet<>(this.userOnlyScripts.get(injectionTime));
  }

  private void updateContentWorldsCreatorScript() {
    String source = generateContentWorldsCreatorCode();
    if (WebViewFeature.isFeatureSupported(WebViewFeature.DOCUMENT_START_SCRIPT)) {
      if (contentWorldsCreatorScript != null) {
        contentWorldsCreatorScript.remove();
      }
      if (!source.isEmpty() && webView != null) {
        contentWorldsCreatorScript = WebViewCompat.addDocumentStartJavaScript(
                webView,
                source,
                new HashSet<String>() {{
                  add("*");
                }}
        );
      }
    }
  }

  public boolean addUserOnlyScript(UserScript userOnlyScript) {
    ContentWorld contentWorld = userOnlyScript.getContentWorld();
    if (contentWorld != null) {
      contentWorlds.add(contentWorld);
    }
    this.updateContentWorldsCreatorScript();
    if (webView != null && userOnlyScript.getInjectionTime() == UserScriptInjectionTime.AT_DOCUMENT_START
            && WebViewFeature.isFeatureSupported(WebViewFeature.DOCUMENT_START_SCRIPT)) {
      ScriptHandler scriptHandler = WebViewCompat.addDocumentStartJavaScript(
              webView,
              wrapSourceCodeInContentWorld(userOnlyScript.getContentWorld(), userOnlyScript.getSource()),
              userOnlyScript.getAllowedOriginRules()
      );
      this.scriptHandlerMap.put(userOnlyScript, scriptHandler);
    }
    return this.userOnlyScripts.get(userOnlyScript.getInjectionTime()).add(userOnlyScript);
  }

  public void addUserOnlyScripts(List<UserScript> userOnlyScripts) {
    for (UserScript userOnlyScript : userOnlyScripts) {
      this.addUserOnlyScript(userOnlyScript);
    }
  }

  public boolean removeUserOnlyScript(UserScript userOnlyScript) {
    if (WebViewFeature.isFeatureSupported(WebViewFeature.DOCUMENT_START_SCRIPT)) {
      ScriptHandler scriptHandler = this.scriptHandlerMap.get(userOnlyScript);
      if (scriptHandler != null) {
        scriptHandler.remove();
        this.scriptHandlerMap.remove(userOnlyScript);
      }
      this.updateContentWorldsCreatorScript();
    }
    return this.userOnlyScripts.get(userOnlyScript.getInjectionTime()).remove(userOnlyScript);
  }

  public boolean removeUserOnlyScriptAt(int index, UserScriptInjectionTime injectionTime) {
    UserScript userOnlyScript = new ArrayList<>(this.userOnlyScripts.get(injectionTime)).get(index);
    return this.removeUserOnlyScript(userOnlyScript);
  }

  public void removeAllUserOnlyScripts() {
    if (WebViewFeature.isFeatureSupported(WebViewFeature.DOCUMENT_START_SCRIPT)) {
      for (UserScript userOnlyScript : this.userOnlyScripts.get(UserScriptInjectionTime.AT_DOCUMENT_START)) {
        ScriptHandler scriptHandler = this.scriptHandlerMap.get(userOnlyScript);
        if (scriptHandler != null) {
          scriptHandler.remove();
          this.scriptHandlerMap.remove(userOnlyScript);
        }
      }
    }
    this.userOnlyScripts.get(UserScriptInjectionTime.AT_DOCUMENT_START).clear();
    this.userOnlyScripts.get(UserScriptInjectionTime.AT_DOCUMENT_END).clear();
  }

  public LinkedHashSet<PluginScript> getPluginScriptsAt(UserScriptInjectionTime injectionTime) {
    return new LinkedHashSet<>(this.pluginScripts.get(injectionTime));
  }

  public LinkedHashSet<PluginScript> getPluginScriptsRequiredInAllContentWorlds() {
    LinkedHashSet<PluginScript> pluginScriptsRequired = new LinkedHashSet<>();
    LinkedHashSet<PluginScript> scripts = this.getPluginScriptsAt(UserScriptInjectionTime.AT_DOCUMENT_START);
    for (PluginScript script : scripts) {
      if (script.isRequiredInAllContentWorlds()) {
        pluginScriptsRequired.add(script);
      }
    }
    return pluginScriptsRequired;
  }

  public boolean addPluginScript(PluginScript pluginScript) {
    ContentWorld contentWorld = pluginScript.getContentWorld();
    if (contentWorld != null) {
      contentWorlds.add(contentWorld);
    }
    this.updateContentWorldsCreatorScript();
    if (webView != null && pluginScript.getInjectionTime() == UserScriptInjectionTime.AT_DOCUMENT_START
            && WebViewFeature.isFeatureSupported(WebViewFeature.DOCUMENT_START_SCRIPT)) {
      ScriptHandler scriptHandler = WebViewCompat.addDocumentStartJavaScript(
              webView,
              wrapSourceCodeInContentWorld(pluginScript.getContentWorld(), pluginScript.getSource()),
              pluginScript.getAllowedOriginRules()
      );
      this.scriptHandlerMap.put(pluginScript, scriptHandler);
    }
    return this.pluginScripts.get(pluginScript.getInjectionTime()).add(pluginScript);
  }

  public void addPluginScripts(List<PluginScript> pluginScripts) {
    for (PluginScript pluginScript : pluginScripts) {
      this.addPluginScript(pluginScript);
    }
  }

  public boolean removePluginScript(PluginScript pluginScript) {
    if (WebViewFeature.isFeatureSupported(WebViewFeature.DOCUMENT_START_SCRIPT)) {
      ScriptHandler scriptHandler = this.scriptHandlerMap.get(pluginScript);
      if (scriptHandler != null) {
        scriptHandler.remove();
        this.scriptHandlerMap.remove(pluginScript);
      }
      this.updateContentWorldsCreatorScript();
    }
    return this.pluginScripts.get(pluginScript.getInjectionTime()).remove(pluginScript);
  }

  public void removeAllPluginScripts() {
    if (WebViewFeature.isFeatureSupported(WebViewFeature.DOCUMENT_START_SCRIPT)) {
      for (PluginScript pluginScript : this.pluginScripts.get(UserScriptInjectionTime.AT_DOCUMENT_START)) {
        ScriptHandler scriptHandler = this.scriptHandlerMap.get(pluginScript);
        if (scriptHandler != null) {
          scriptHandler.remove();
          this.scriptHandlerMap.remove(pluginScript);
        }
      }
    }
    this.pluginScripts.get(UserScriptInjectionTime.AT_DOCUMENT_START).clear();
    this.pluginScripts.get(UserScriptInjectionTime.AT_DOCUMENT_END).clear();
  }

  public LinkedHashSet<UserScript> getUserOnlyScriptAsList() {
    LinkedHashSet<UserScript> userOnlyScripts = new LinkedHashSet<>();
    Collection<LinkedHashSet<UserScript>> collection = this.userOnlyScripts.values();
    for (LinkedHashSet<UserScript> list : collection) {
      userOnlyScripts.addAll(list);
    }
    return userOnlyScripts;
  }

  public LinkedHashSet<PluginScript> getPluginScriptAsList() {
    LinkedHashSet<PluginScript> pluginScripts = new LinkedHashSet<>();
    Collection<LinkedHashSet<PluginScript>> collection = this.pluginScripts.values();
    for (LinkedHashSet<PluginScript> list : collection) {
      pluginScripts.addAll(list);
    }
    return pluginScripts;
  }

  public void resetContentWorlds() {
    this.contentWorlds.clear();
    this.contentWorlds.add(ContentWorld.PAGE);

    LinkedHashSet<PluginScript> pluginScripts = this.getPluginScriptAsList();
    for (PluginScript pluginScript : pluginScripts) {
      ContentWorld contentWorld = pluginScript.getContentWorld();
      this.contentWorlds.add(contentWorld);
    }

    LinkedHashSet<UserScript> userOnlyScripts = this.getUserOnlyScriptAsList();
    for (UserScript userOnlyScript : userOnlyScripts) {
      ContentWorld contentWorld = userOnlyScript.getContentWorld();
      this.contentWorlds.add(contentWorld);
    }
  }

  public boolean containsPluginScript(PluginScript pluginScript) {
    return this.getPluginScriptAsList().contains(pluginScript);
  }

  public boolean containsPluginScriptByGroupName(String groupName) {
    LinkedHashSet<PluginScript> pluginScripts = this.getPluginScriptAsList();
    for (PluginScript pluginScript : pluginScripts) {
      if (Util.objEquals(groupName, pluginScript.getGroupName())) {
        return true;
      }
    }
    return false;
  }

  public boolean containsUserOnlyScript(UserScript userOnlyScript) {
    return this.getUserOnlyScriptAsList().contains(userOnlyScript);
  }

  public boolean containsUserOnlyScriptByGroupName(String groupName) {
    LinkedHashSet<UserScript> userOnlyScripts = this.getUserOnlyScriptAsList();
    for (UserScript userOnlyScript : userOnlyScripts) {
      if (Util.objEquals(groupName, userOnlyScript.getGroupName())) {
        return true;
      }
    }

    return false;
  }

  public void removePluginScriptsByGroupName(String groupName) {
    LinkedHashSet<PluginScript> pluginScripts = this.getPluginScriptAsList();
    for (PluginScript pluginScript : pluginScripts) {
      if (Util.objEquals(groupName, pluginScript.getGroupName())) {
        this.removePluginScript(pluginScript);
      }
    }
  }

  public void removeUserOnlyScriptsByGroupName(String groupName) {
    LinkedHashSet<UserScript> userOnlyScripts = this.getUserOnlyScriptAsList();
    for (UserScript userOnlyScript : userOnlyScripts) {
      if (Util.objEquals(groupName, userOnlyScript.getGroupName())) {
        this.removeUserOnlyScript(userOnlyScript);
      }
    }
  }

  @NonNull
  public LinkedHashSet<ContentWorld> getContentWorlds() {
    return new LinkedHashSet<>(this.contentWorlds);
  }

  private static final String USER_SCRIPTS_AT_DOCUMENT_START_WRAPPER_JS_SOURCE = "if (window." + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + " != null && (window." + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + "._userScriptsAtDocumentStartLoaded == null || !window." + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + "._userScriptsAtDocumentStartLoaded)) {" +
          "  window." + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + "._userScriptsAtDocumentStartLoaded = true;" +
          "  " + PluginScriptsUtil.VAR_PLACEHOLDER_VALUE +
          "}";

  private static final String USER_SCRIPTS_AT_DOCUMENT_END_WRAPPER_JS_SOURCE = "if (window." + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + " != null && (window." + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + "._userScriptsAtDocumentEndLoaded == null || !window." + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + "._userScriptsAtDocumentEndLoaded)) {" +
          "  window." + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + "._userScriptsAtDocumentEndLoaded = true;" +
          "  " + PluginScriptsUtil.VAR_PLACEHOLDER_VALUE +
          "}";

  private static final String CONTENT_WORLDS_GENERATOR_JS_SOURCE = "(function() {" +
          "  var interval = setInterval(function() {" +
          "    if (document.body == null) {return;}" +
          "    var contentWorldNames = [" + PluginScriptsUtil.VAR_CONTENT_WORLD_NAME_ARRAY + "];" +
          "    for (var contentWorldName of contentWorldNames) {" +
          "      var iframeId = '" + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + "_' + contentWorldName;" +
          "      var iframe = document.getElementById(iframeId);" +
          "      if (iframe == null) {" +
          "        iframe = document.createElement('iframe');" +
          "        iframe.id = iframeId;" +
          "        iframe.style = 'display: none; z-index: 0; position: absolute; width: 0px; height: 0px';" +
          "        document.body.append(iframe);" +
          "      }" +
          "      if (iframe.contentWindow.document.getElementById('" + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + "_plugin_scripts') == null) {" +
          "        var script = iframe.contentWindow.document.createElement('script');" +
          "        script.id = '" + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + "_plugin_scripts';" +
          "        script.innerHTML = " + PluginScriptsUtil.VAR_JSON_SOURCE_ENCODED + ";" +
          "        iframe.contentWindow.document.body.append(script);" +
          "      }" +
          "    }" +
          "    clearInterval(interval);" +
          "  });" +
          "})();";

  private static final String CONTENT_WORLD_WRAPPER_JS_SOURCE = "(function() {" +
          "  var interval = setInterval(function() {" +
          "    if (document.body == null) {return;}" +
          "    var iframeId = '" + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + "_" + PluginScriptsUtil.VAR_CONTENT_WORLD_NAME + "';" +
          "    var iframe = document.getElementById(iframeId);" +
          "    if (iframe == null) {" +
          "      iframe = document.createElement('iframe');" +
          "      iframe.id = iframeId;" +
          "      iframe.style = 'display: none; z-index: 0; position: absolute; width: 0px; height: 0px';" +
          "      document.body.append(iframe);" +
          "    }" +
          "    if (iframe.contentWindow.document.querySelector('#" + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + "_plugin_scripts') == null) {" +
          "      return;" +
          "    }" +
          "    var script = iframe.contentWindow.document.createElement('script');" +
          "    script.innerHTML = " + PluginScriptsUtil.VAR_JSON_SOURCE_ENCODED + ";" +
          "    iframe.contentWindow.document.body.append(script);" +
          "    clearInterval(interval);" +
          "  });" +
          "})();";

  private static final String DOCUMENT_READY_WRAPPER_JS_SOURCE = "if (document.readyState === 'interactive' || document.readyState === 'complete') { " +
          "  " + PluginScriptsUtil.VAR_PLACEHOLDER_VALUE +
          "}";

  @Override
  public void dispose() {
    if (WebViewFeature.isFeatureSupported(WebViewFeature.DOCUMENT_START_SCRIPT) && contentWorldsCreatorScript != null) {
      contentWorldsCreatorScript.remove();
    }
    removeAllUserOnlyScripts();
    removeAllPluginScripts();
    webView = null;
  }
}
