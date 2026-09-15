package com.pichillilorenzo.flutter_inappwebview_android.types;

public class ServerTrustChallenge extends URLAuthenticationChallenge {
  public ServerTrustChallenge(URLProtectionSpace protectionSpace) {
    super(protectionSpace);
  }

  @Override
  public String toString() {
    return "ServerTrustChallenge{} " + super.toString();
  }
}
