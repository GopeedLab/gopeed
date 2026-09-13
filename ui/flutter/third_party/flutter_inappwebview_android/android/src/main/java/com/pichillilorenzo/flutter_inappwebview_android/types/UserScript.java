package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

public class UserScript {
  @Nullable
  private String groupName;
  @NonNull
  private String source;
  @NonNull
  private UserScriptInjectionTime injectionTime;
  @NonNull
  private ContentWorld contentWorld;
  @NonNull
  private Set<String> allowedOriginRules = new HashSet<>();

  public UserScript(@Nullable String groupName, @NonNull String source,
                    @NonNull UserScriptInjectionTime injectionTime, @Nullable ContentWorld contentWorld,
                    @Nullable Set<String> allowedOriginRules) {
    this.groupName = groupName;
    this.source = source;
    this.injectionTime = injectionTime;
    this.contentWorld = contentWorld == null ? ContentWorld.PAGE : contentWorld;
    this.allowedOriginRules = allowedOriginRules == null ? new HashSet<String>() {{
      add("*");
    }} : allowedOriginRules;
  }

  @Nullable
  public static UserScript fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    String groupName = (String) map.get("groupName");
    String source = (String) map.get("source");
    UserScriptInjectionTime injectionTime = UserScriptInjectionTime.fromValue((int) map.get("injectionTime"));
    ContentWorld contentWorld = ContentWorld.fromMap((Map<String, Object>) map.get("contentWorld"));
    Set<String> allowedOriginRules = new HashSet<>((List<String>) map.get("allowedOriginRules"));
    assert source != null;
    return new UserScript(groupName, source, injectionTime, contentWorld, allowedOriginRules);
  }

  @Nullable
  public String getGroupName() {
    return groupName;
  }

  public void setGroupName(@Nullable String groupName) {
    this.groupName = groupName;
  }

  @NonNull
  public String getSource() {
    return source;
  }

  public void setSource(@NonNull String source) {
    this.source = source;
  }

  @NonNull
  public UserScriptInjectionTime getInjectionTime() {
    return injectionTime;
  }

  public void setInjectionTime(@NonNull UserScriptInjectionTime injectionTime) {
    this.injectionTime = injectionTime;
  }

  @NonNull
  public ContentWorld getContentWorld() {
    return contentWorld;
  }

  public void setContentWorld(@Nullable ContentWorld contentWorld) {
    this.contentWorld = contentWorld == null ? ContentWorld.PAGE : contentWorld;
  }

  @NonNull
  public Set<String> getAllowedOriginRules() {
    return allowedOriginRules;
  }

  public void setAllowedOriginRules(@NonNull Set<String> allowedOriginRules) {
    this.allowedOriginRules = allowedOriginRules;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    UserScript that = (UserScript) o;

    if (groupName != null ? !groupName.equals(that.groupName) : that.groupName != null)
      return false;
    if (!source.equals(that.source)) return false;
    if (injectionTime != that.injectionTime) return false;
    if (!contentWorld.equals(that.contentWorld)) return false;
    return allowedOriginRules.equals(that.allowedOriginRules);
  }

  @Override
  public int hashCode() {
    int result = groupName != null ? groupName.hashCode() : 0;
    result = 31 * result + source.hashCode();
    result = 31 * result + injectionTime.hashCode();
    result = 31 * result + contentWorld.hashCode();
    result = 31 * result + allowedOriginRules.hashCode();
    return result;
  }

  @Override
  public String toString() {
    return "UserScript{" +
            "groupName='" + groupName + '\'' +
            ", source='" + source + '\'' +
            ", injectionTime=" + injectionTime +
            ", contentWorld=" + contentWorld +
            ", allowedOriginRules=" + allowedOriginRules +
            '}';
  }
}
