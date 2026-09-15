package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.Nullable;

import java.util.Map;

public class JsBeforeUnloadResponse {
  private String message;
  private String confirmButtonTitle;
  private String cancelButtonTitle;
  private boolean handledByClient;
  @Nullable
  private Integer action;

  public JsBeforeUnloadResponse(String message, String confirmButtonTitle, String cancelButtonTitle, boolean handledByClient, @Nullable Integer action) {
    this.message = message;
    this.confirmButtonTitle = confirmButtonTitle;
    this.cancelButtonTitle = cancelButtonTitle;
    this.handledByClient = handledByClient;
    this.action = action;
  }

  @Nullable
  public static JsBeforeUnloadResponse fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    String message = (String) map.get("message");
    String confirmButtonTitle = (String) map.get("confirmButtonTitle");
    String cancelButtonTitle = (String) map.get("cancelButtonTitle");
    boolean handledByClient = (boolean) map.get("handledByClient");
    Integer action = (Integer) map.get("action");
    return new JsBeforeUnloadResponse(message, confirmButtonTitle, cancelButtonTitle, handledByClient, action);
  }

  public String getMessage() {
    return message;
  }

  public void setMessage(String message) {
    this.message = message;
  }

  public String getConfirmButtonTitle() {
    return confirmButtonTitle;
  }

  public void setConfirmButtonTitle(String confirmButtonTitle) {
    this.confirmButtonTitle = confirmButtonTitle;
  }

  public String getCancelButtonTitle() {
    return cancelButtonTitle;
  }

  public void setCancelButtonTitle(String cancelButtonTitle) {
    this.cancelButtonTitle = cancelButtonTitle;
  }

  public boolean isHandledByClient() {
    return handledByClient;
  }

  public void setHandledByClient(boolean handledByClient) {
    this.handledByClient = handledByClient;
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

    JsBeforeUnloadResponse that = (JsBeforeUnloadResponse) o;

    if (handledByClient != that.handledByClient) return false;
    if (message != null ? !message.equals(that.message) : that.message != null) return false;
    if (confirmButtonTitle != null ? !confirmButtonTitle.equals(that.confirmButtonTitle) : that.confirmButtonTitle != null)
      return false;
    if (cancelButtonTitle != null ? !cancelButtonTitle.equals(that.cancelButtonTitle) : that.cancelButtonTitle != null)
      return false;
    return action != null ? action.equals(that.action) : that.action == null;
  }

  @Override
  public int hashCode() {
    int result = message != null ? message.hashCode() : 0;
    result = 31 * result + (confirmButtonTitle != null ? confirmButtonTitle.hashCode() : 0);
    result = 31 * result + (cancelButtonTitle != null ? cancelButtonTitle.hashCode() : 0);
    result = 31 * result + (handledByClient ? 1 : 0);
    result = 31 * result + (action != null ? action.hashCode() : 0);
    return result;
  }

  @Override
  public String toString() {
    return "JsConfirmResponse{" +
            "message='" + message + '\'' +
            ", confirmButtonTitle='" + confirmButtonTitle + '\'' +
            ", cancelButtonTitle='" + cancelButtonTitle + '\'' +
            ", handledByClient=" + handledByClient +
            ", action=" + action +
            '}';
  }
}
