package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.Nullable;

import java.util.Map;

public class ServerTrustAuthResponse {
  @Nullable
  private Integer action;

  public ServerTrustAuthResponse(@Nullable Integer action) {
    this.action = action;
  }

  @Nullable
  public static ServerTrustAuthResponse fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    Integer action = (Integer) map.get("action");
    return new ServerTrustAuthResponse(action);
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

    ServerTrustAuthResponse that = (ServerTrustAuthResponse) o;

    return action != null ? action.equals(that.action) : that.action == null;
  }

  @Override
  public int hashCode() {
    return action != null ? action.hashCode() : 0;
  }

  @Override
  public String toString() {
    return "ServerTrustAuthResponse{" +
            "action=" + action +
            '}';
  }
}
