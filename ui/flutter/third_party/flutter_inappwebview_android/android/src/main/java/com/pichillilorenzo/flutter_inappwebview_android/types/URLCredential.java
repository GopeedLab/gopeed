package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import java.util.HashMap;
import java.util.Map;

public class URLCredential {
  @Nullable
  private Long id;
  @Nullable
  private String username;
  @Nullable
  private String password;
  @Nullable
  private Long protectionSpaceId;

  public URLCredential(@Nullable String username, @Nullable String password) {
    this.username = username;
    this.password = password;
  }

  public URLCredential (@Nullable Long id, @NonNull String username, @NonNull String password, @Nullable Long protectionSpaceId) {
    this.id = id;
    this.username = username;
    this.password = password;
    this.protectionSpaceId = protectionSpaceId;
  }

  public Map<String, Object> toMap() {
    Map<String, Object> urlCredentialMap = new HashMap<>();
    urlCredentialMap.put("username", username);
    urlCredentialMap.put("password", password);
    urlCredentialMap.put("certificates", null);
    urlCredentialMap.put("persistence", null);
    return urlCredentialMap;
  }

  @Nullable
  public Long getId() {
    return id;
  }

  public void setId(@Nullable Long id) {
    this.id = id;
  }

  @Nullable
  public String getUsername() {
    return username;
  }

  public void setUsername(@Nullable String username) {
    this.username = username;
  }

  @Nullable
  public String getPassword() {
    return password;
  }

  public void setPassword(@Nullable String password) {
    this.password = password;
  }

  @Nullable
  public Long getProtectionSpaceId() {
    return protectionSpaceId;
  }

  public void setProtectionSpaceId(@Nullable Long protectionSpaceId) {
    this.protectionSpaceId = protectionSpaceId;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    URLCredential that = (URLCredential) o;

    if (username != null ? !username.equals(that.username) : that.username != null) return false;
    return password != null ? password.equals(that.password) : that.password == null;
  }

  @Override
  public int hashCode() {
    int result = username != null ? username.hashCode() : 0;
    result = 31 * result + (password != null ? password.hashCode() : 0);
    return result;
  }

  @Override
  public String toString() {
    return "URLCredential{" +
            "username='" + username + '\'' +
            ", password='" + password + '\'' +
            '}';
  }
}
