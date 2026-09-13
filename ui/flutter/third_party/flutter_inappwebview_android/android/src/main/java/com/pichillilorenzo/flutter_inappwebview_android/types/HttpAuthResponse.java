package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import java.util.Map;

public class HttpAuthResponse {
  @NonNull
  private String username;
  @NonNull
  private String password;
  boolean permanentPersistence;
  @Nullable
  private Integer action;

  public HttpAuthResponse(@NonNull String username, @NonNull String password, boolean permanentPersistence, @Nullable Integer action) {
    this.username = username;
    this.password = password;
    this.permanentPersistence = permanentPersistence;
    this.action = action;
  }

  @Nullable
  public static HttpAuthResponse fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    String username = (String) map.get("username");
    String password = (String) map.get("password");
    boolean permanentPersistence = (boolean) map.get("permanentPersistence");
    Integer action = (Integer) map.get("action");
    return new HttpAuthResponse(username, password, permanentPersistence, action);
  }

  @NonNull
  public String getUsername() {
    return username;
  }

  public void setUsername(@NonNull String username) {
    this.username = username;
  }

  @NonNull
  public String getPassword() {
    return password;
  }

  public void setPassword(@NonNull String password) {
    this.password = password;
  }

  public boolean isPermanentPersistence() {
    return permanentPersistence;
  }

  public void setPermanentPersistence(boolean permanentPersistence) {
    this.permanentPersistence = permanentPersistence;
  }

  @Nullable
  public Integer getAction() {
    return action;
  }

  public void setAction(@Nullable Integer action) {
    this.action = action;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    HttpAuthResponse that = (HttpAuthResponse) o;

    if (permanentPersistence != that.permanentPersistence) return false;
    if (!username.equals(that.username)) return false;
    if (!password.equals(that.password)) return false;
    return action != null ? action.equals(that.action) : that.action == null;
  }

  @Override
  public int hashCode() {
    int result = username.hashCode();
    result = 31 * result + password.hashCode();
    result = 31 * result + (permanentPersistence ? 1 : 0);
    result = 31 * result + (action != null ? action.hashCode() : 0);
    return result;
  }

  @Override
  public String toString() {
    return "HttpAuthResponse{" +
            "username='" + username + '\'' +
            ", password='" + password + '\'' +
            ", permanentPersistence=" + permanentPersistence +
            ", action=" + action +
            '}';
  }
}
