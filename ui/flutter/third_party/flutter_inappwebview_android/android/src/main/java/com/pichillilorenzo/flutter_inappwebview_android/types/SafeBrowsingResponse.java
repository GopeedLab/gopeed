package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.Nullable;

import java.util.Map;

public class SafeBrowsingResponse {
  private boolean report;
  @Nullable
  private Integer action;

  public SafeBrowsingResponse(boolean report, @Nullable Integer action) {
    this.report = report;
    this.action = action;
  }

  @Nullable
  public static SafeBrowsingResponse fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    boolean report = (boolean) map.get("report");
    Integer action = (Integer) map.get("action");
    return new SafeBrowsingResponse(report, action);
  }

  public boolean isReport() {
    return report;
  }

  public void setReport(boolean report) {
    this.report = report;
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

    SafeBrowsingResponse that = (SafeBrowsingResponse) o;

    if (report != that.report) return false;
    return action != null ? action.equals(that.action) : that.action == null;
  }

  @Override
  public int hashCode() {
    int result = (report ? 1 : 0);
    result = 31 * result + (action != null ? action.hashCode() : 0);
    return result;
  }

  @Override
  public String toString() {
    return "SafeBrowsingResponse{" +
            "report=" + report +
            ", action=" + action +
            '}';
  }
}
