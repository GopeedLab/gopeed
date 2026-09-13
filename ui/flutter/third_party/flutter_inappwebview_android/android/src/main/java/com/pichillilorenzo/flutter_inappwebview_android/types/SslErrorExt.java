package com.pichillilorenzo.flutter_inappwebview_android.types;

import android.net.http.SslCertificate;
import android.net.http.SslError;

import androidx.annotation.Nullable;

import java.util.HashMap;
import java.util.Map;

public class SslErrorExt extends SslError {

  private SslErrorExt(int error, SslCertificate certificate, String url) {
    super(error, certificate, url);
  }

  @Nullable
  static public Map<String, Object> toMap(SslError sslError) {
    if (sslError == null) {
      return null;
    }

    int primaryError = sslError.getPrimaryError();

    String message;
    switch (primaryError) {
      case SslError.SSL_DATE_INVALID:
        message = "The date of the certificate is invalid";
        break;
      case SslError.SSL_EXPIRED:
        message = "The certificate has expired";
        break;
      case SslError.SSL_IDMISMATCH:
        message = "Hostname mismatch";
        break;
      case SslError.SSL_INVALID:
        message = "A generic error occurred";
        break;
      case SslError.SSL_NOTYETVALID:
        message = "The certificate is not yet valid";
        break;
      case SslError.SSL_UNTRUSTED:
        message = "The certificate authority is not trusted";
        break;
      default:
        message = null;
        break;
    }

    Map<String, Object> urlProtectionSpaceMap = new HashMap<>();
    urlProtectionSpaceMap.put("code", primaryError >= 0 ? primaryError : null);
    urlProtectionSpaceMap.put("message", message);
    return urlProtectionSpaceMap;
  }

}
