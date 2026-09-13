package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.Nullable;

import java.util.Map;

public class HttpAuthenticationChallenge extends URLAuthenticationChallenge {
  private int previousFailureCount;
  @Nullable
  URLCredential proposedCredential;

  public HttpAuthenticationChallenge(URLProtectionSpace protectionSpace, int previousFailureCount, @Nullable URLCredential proposedCredential) {
    super(protectionSpace);
    this.previousFailureCount = previousFailureCount;
    this.proposedCredential = proposedCredential;
  }

  public Map<String, Object> toMap() {
    Map<String, Object> challengeMap = super.toMap();
    challengeMap.put("previousFailureCount", previousFailureCount);
    challengeMap.put("proposedCredential", (proposedCredential != null) ? proposedCredential.toMap() : null);
    challengeMap.put("failureResponse", null);
    challengeMap.put("error", null);
    return challengeMap;
  }

  public int getPreviousFailureCount() {
    return previousFailureCount;
  }

  public void setPreviousFailureCount(int previousFailureCount) {
    this.previousFailureCount = previousFailureCount;
  }

  @Nullable
  public URLCredential getProposedCredential() {
    return proposedCredential;
  }

  public void setProposedCredential(@Nullable URLCredential proposedCredential) {
    this.proposedCredential = proposedCredential;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;
    if (!super.equals(o)) return false;

    HttpAuthenticationChallenge that = (HttpAuthenticationChallenge) o;

    if (previousFailureCount != that.previousFailureCount) return false;
    return proposedCredential != null ? proposedCredential.equals(that.proposedCredential) : that.proposedCredential == null;
  }

  @Override
  public int hashCode() {
    int result = super.hashCode();
    result = 31 * result + previousFailureCount;
    result = 31 * result + (proposedCredential != null ? proposedCredential.hashCode() : 0);
    return result;
  }

  @Override
  public String toString() {
    return "HttpAuthenticationChallenge{" +
            "previousFailureCount=" + previousFailureCount +
            ", proposedCredential=" + proposedCredential +
            "} " + super.toString();
  }
}
