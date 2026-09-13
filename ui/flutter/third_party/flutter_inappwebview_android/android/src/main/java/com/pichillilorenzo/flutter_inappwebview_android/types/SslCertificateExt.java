package com.pichillilorenzo.flutter_inappwebview_android.types;

import android.net.http.SslCertificate;
import android.os.Build;

import androidx.annotation.Nullable;

import com.pichillilorenzo.flutter_inappwebview_android.Util;

import java.security.cert.CertificateEncodingException;
import java.security.cert.X509Certificate;
import java.util.HashMap;
import java.util.Map;

public class SslCertificateExt extends SslCertificate {

  private SslCertificateExt(X509Certificate certificate) {
    super(certificate);
  }

  @Nullable
  static public Map<String, Object> toMap(@Nullable SslCertificate sslCertificate) {
    if (sslCertificate == null) {
      return null;
    }

    DName issuedByName = sslCertificate.getIssuedBy();
    Map<String, Object> issuedBy = null;
    if (issuedByName != null) {
      issuedBy = new HashMap<>();
      issuedBy.put("CName", issuedByName.getCName());
      issuedBy.put("DName", issuedByName.getDName());
      issuedBy.put("OName", issuedByName.getOName());
      issuedBy.put("UName", issuedByName.getUName());
    }

    DName issuedToName = sslCertificate.getIssuedTo();
    Map<String, Object> issuedTo = null;
    if (issuedToName != null) {
      issuedTo = new HashMap<>();
      issuedTo.put("CName", issuedToName.getCName());
      issuedTo.put("DName", issuedToName.getDName());
      issuedTo.put("OName", issuedToName.getOName());
      issuedTo.put("UName", issuedToName.getUName());
    }

    byte[] x509CertificateData = null;

    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
      try {
        X509Certificate certificate = sslCertificate.getX509Certificate();
        if (certificate != null) {
          x509CertificateData = certificate.getEncoded();
        }
      } catch (CertificateEncodingException e) {
        e.printStackTrace();
      }
    } else {
      try {
        x509CertificateData = Util.getX509CertFromSslCertHack(sslCertificate).getEncoded();
      } catch (CertificateEncodingException e) {
        e.printStackTrace();
      }
    }

    long validNotAfterDate = sslCertificate.getValidNotAfterDate().getTime();
    long validNotBeforeDate = sslCertificate.getValidNotBeforeDate().getTime();

    Map<String, Object> sslCertificateMap = new HashMap<>();
    sslCertificateMap.put("issuedBy", issuedBy);
    sslCertificateMap.put("issuedTo", issuedTo);
    sslCertificateMap.put("validNotAfterDate", validNotAfterDate);
    sslCertificateMap.put("validNotBeforeDate", validNotBeforeDate);
    sslCertificateMap.put("x509Certificate", x509CertificateData);
    return sslCertificateMap;
  }
}
