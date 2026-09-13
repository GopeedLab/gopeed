package com.pichillilorenzo.flutter_inappwebview_android.types;

import java.util.Map;

public class CreateWindowAction extends NavigationAction {
  int windowId;
  boolean isDialog;

  public CreateWindowAction(URLRequest request, boolean isForMainFrame, boolean hasGesture, boolean isRedirect, int windowId, boolean isDialog) {
    super(request, isForMainFrame, hasGesture, isRedirect);
    this.windowId = windowId;
    this.isDialog = isDialog;
  }

  public Map<String, Object> toMap() {
    Map<String, Object> createWindowActionMap = super.toMap();
    createWindowActionMap.put("windowId", windowId);
    createWindowActionMap.put("isDialog", isDialog);
    createWindowActionMap.put("windowFeatures", null);
    return createWindowActionMap;
  }

  public int getWindowId() {
    return windowId;
  }

  public void setWindowId(int windowId) {
    this.windowId = windowId;
  }

  public boolean isDialog() {
    return isDialog;
  }

  public void setDialog(boolean dialog) {
    isDialog = dialog;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;
    if (!super.equals(o)) return false;

    CreateWindowAction that = (CreateWindowAction) o;

    if (windowId != that.windowId) return false;
    return isDialog == that.isDialog;
  }

  @Override
  public int hashCode() {
    int result = super.hashCode();
    result = 31 * result + windowId;
    result = 31 * result + (isDialog ? 1 : 0);
    return result;
  }

  @Override
  public String toString() {
    return "CreateWindowAction{" +
            "windowId=" + windowId +
            ", isDialog=" + isDialog +
            ", request=" + request +
            ", isForMainFrame=" + isForMainFrame +
            ", hasGesture=" + hasGesture +
            ", isRedirect=" + isRedirect +
            '}';
  }
}
