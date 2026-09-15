package com.pichillilorenzo.flutter_inappwebview_android.types;

import java.util.HashMap;
import java.util.Map;

public class URLAuthenticationChallenge {
  private URLProtectionSpace protectionSpace;

  public URLAuthenticationChallenge(URLProtectionSpace protectionSpace) {
    this.protectionSpace = protectionSpace;
  }

  public Map<String, Object> toMap() {
    Map<String, Object> challengeMap = new HashMap<>();
    challengeMap.put("protectionSpace", protectionSpace.toMap());
    return challengeMap;
  }

  public URLProtectionSpace getProtectionSpace() {
    return protectionSpace;
  }

  public void setProtectionSpace(URLProtectionSpace protectionSpace) {
    this.protectionSpace = protectionSpace;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    URLAuthenticationChallenge challenge = (URLAuthenticationChallenge) o;

    return protectionSpace.equals(challenge.protectionSpace);
  }

  @Override
  public int hashCode() {
    return protectionSpace.hashCode();
  }

  @Override
  public String toString() {
    return "URLAuthenticationChallenge{" +
            "protectionSpace=" + protectionSpace +
            '}';
  }
}
