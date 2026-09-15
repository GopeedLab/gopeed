package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.Nullable;

import java.security.Principal;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.Map;

public class ClientCertChallenge extends URLAuthenticationChallenge {
  @Nullable
  private Principal[] principals;
  @Nullable
  private String[] keyTypes;

  public ClientCertChallenge(URLProtectionSpace protectionSpace, @Nullable Principal[] principals, @Nullable String[] keyTypes) {
    super(protectionSpace);
    this.principals = principals;
    this.keyTypes = keyTypes;
  }

  public Map<String, Object> toMap() {
    List<String> principalList = null;
    if (principals != null) {
      principalList = new ArrayList<>();
      for (Principal principal : principals) {
        principalList.add(principal.getName());
      }
    }

    Map<String, Object> challengeMap = super.toMap();
    challengeMap.put("principals", principalList);
    challengeMap.put("keyTypes", keyTypes != null ? Arrays.asList(keyTypes) : null);
    return challengeMap;
  }

  @Nullable
  public Principal[] getPrincipals() {
    return principals;
  }

  public void setPrincipals(@Nullable Principal[] principals) {
    this.principals = principals;
  }

  @Nullable
  public String[] getKeyTypes() {
    return keyTypes;
  }

  public void setKeyTypes(@Nullable String[] keyTypes) {
    this.keyTypes = keyTypes;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;
    if (!super.equals(o)) return false;

    ClientCertChallenge that = (ClientCertChallenge) o;

    // Probably incorrect - comparing Object[] arrays with Arrays.equals
    if (!Arrays.equals(principals, that.principals)) return false;
    // Probably incorrect - comparing Object[] arrays with Arrays.equals
    return Arrays.equals(keyTypes, that.keyTypes);
  }

  @Override
  public int hashCode() {
    int result = super.hashCode();
    result = 31 * result + Arrays.hashCode(principals);
    result = 31 * result + Arrays.hashCode(keyTypes);
    return result;
  }

  @Override
  public String toString() {
    return "ClientCertChallenge{" +
            "principals=" + Arrays.toString(principals) +
            ", keyTypes=" + Arrays.toString(keyTypes) +
            "} " + super.toString();
  }
}
