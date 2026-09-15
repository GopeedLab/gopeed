package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import java.util.Map;

public class ClientCertResponse {
  @NonNull
  private String certificatePath;
  @Nullable
  private String certificatePassword;
  @NonNull
  private String keyStoreType;
  @Nullable
  private Integer action;

  public ClientCertResponse(@NonNull String certificatePath, @Nullable String certificatePassword, @NonNull String keyStoreType, @Nullable Integer action) {
    this.certificatePath = certificatePath;
    this.certificatePassword = certificatePassword;
    this.keyStoreType = keyStoreType;
    this.action = action;
  }

  @Nullable
  public static ClientCertResponse fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    String certificatePath = (String) map.get("certificatePath");
    String certificatePassword = (String) map.get("certificatePassword");
    String keyStoreType = (String) map.get("keyStoreType");
    Integer action = (Integer) map.get("action");
    return new ClientCertResponse(certificatePath, certificatePassword, keyStoreType, action);
  }

  @NonNull
  public String getCertificatePath() {
    return certificatePath;
  }

  public void setCertificatePath(@NonNull String certificatePath) {
    this.certificatePath = certificatePath;
  }

  @Nullable
  public String getCertificatePassword() {
    return certificatePassword;
  }

  public void setCertificatePassword(@Nullable String certificatePassword) {
    this.certificatePassword = certificatePassword;
  }

  @NonNull
  public String getKeyStoreType() {
    return keyStoreType;
  }

  public void setKeyStoreType(@NonNull String keyStoreType) {
    this.keyStoreType = keyStoreType;
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

    ClientCertResponse that = (ClientCertResponse) o;

    if (!certificatePath.equals(that.certificatePath)) return false;
    if (certificatePassword != null ? !certificatePassword.equals(that.certificatePassword) : that.certificatePassword != null)
      return false;
    if (!keyStoreType.equals(that.keyStoreType)) return false;
    return action != null ? action.equals(that.action) : that.action == null;
  }

  @Override
  public int hashCode() {
    int result = certificatePath.hashCode();
    result = 31 * result + (certificatePassword != null ? certificatePassword.hashCode() : 0);
    result = 31 * result + keyStoreType.hashCode();
    result = 31 * result + (action != null ? action.hashCode() : 0);
    return result;
  }

  @Override
  public String toString() {
    return "ClientCertResponse{" +
            "certificatePath='" + certificatePath + '\'' +
            ", certificatePassword='" + certificatePassword + '\'' +
            ", keyStoreType='" + keyStoreType + '\'' +
            ", action=" + action +
            '}';
  }
}
