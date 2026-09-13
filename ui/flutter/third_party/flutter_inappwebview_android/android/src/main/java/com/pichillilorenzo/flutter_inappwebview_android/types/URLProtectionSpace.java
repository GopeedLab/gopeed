package com.pichillilorenzo.flutter_inappwebview_android.types;

import android.net.http.SslCertificate;
import android.net.http.SslError;

import androidx.annotation.Nullable;

import java.util.HashMap;
import java.util.Map;

public class URLProtectionSpace {
  @Nullable
  private Long id;
  private String host;
  private String protocol;
  @Nullable
  private String realm;
  private int port;
  @Nullable
  private SslCertificate sslCertificate;
  @Nullable
  private SslError sslError;

  public URLProtectionSpace(String host, String protocol, @Nullable String realm, int port, @Nullable SslCertificate sslCertificate, @Nullable SslError sslError) {
    this.host = host;
    this.protocol = protocol;
    this.realm = realm;
    this.port = port;
    this.sslCertificate = sslCertificate;
    this.sslError = sslError;
  }

  public URLProtectionSpace(@Nullable Long id, String host, String protocol, @Nullable String realm, int port) {
    this.id = id;
    this.host = host;
    this.protocol = protocol;
    this.realm = realm;
    this.port = port;
  }

  public Map<String, Object> toMap() {
    Map<String, Object> urlProtectionSpaceMap = new HashMap<>();
    urlProtectionSpaceMap.put("host", host);
    urlProtectionSpaceMap.put("protocol", protocol);
    urlProtectionSpaceMap.put("realm", realm);
    urlProtectionSpaceMap.put("port", port);
    urlProtectionSpaceMap.put("sslCertificate", SslCertificateExt.toMap(sslCertificate));
    urlProtectionSpaceMap.put("sslError", SslErrorExt.toMap(sslError));
    urlProtectionSpaceMap.put("authenticationMethod", null);
    urlProtectionSpaceMap.put("distinguishedNames", null);
    urlProtectionSpaceMap.put("receivesCredentialSecurely", null);
    urlProtectionSpaceMap.put("isProxy", null);
    urlProtectionSpaceMap.put("proxyType", null);
    return urlProtectionSpaceMap;
  }

  @Nullable
  public Long getId() {
    return id;
  }

  public void setId(@Nullable Long id) {
    this.id = id;
  }
  
  public String getHost() {
    return host;
  }

  public void setHost(String host) {
    this.host = host;
  }

  public String getProtocol() {
    return protocol;
  }

  public void setProtocol(String protocol) {
    this.protocol = protocol;
  }

  @Nullable
  public String getRealm() {
    return realm;
  }

  public void setRealm(@Nullable String realm) {
    this.realm = realm;
  }

  public int getPort() {
    return port;
  }

  public void setPort(int port) {
    this.port = port;
  }

  @Nullable
  public SslCertificate getSslCertificate() {
    return sslCertificate;
  }

  public void setSslCertificate(@Nullable SslCertificate sslCertificateExt) {
    this.sslCertificate = sslCertificateExt;
  }

  @Nullable
  public SslError getSslError() {
    return sslError;
  }

  public void setSslError(@Nullable SslError sslError) {
    this.sslError = sslError;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    URLProtectionSpace that = (URLProtectionSpace) o;

    if (port != that.port) return false;
    if (!host.equals(that.host)) return false;
    if (!protocol.equals(that.protocol)) return false;
    if (realm != null ? !realm.equals(that.realm) : that.realm != null) return false;
    if (sslCertificate != null ? !sslCertificate.equals(that.sslCertificate) : that.sslCertificate != null)
      return false;
    return sslError != null ? sslError.equals(that.sslError) : that.sslError == null;
  }

  @Override
  public int hashCode() {
    int result = host.hashCode();
    result = 31 * result + protocol.hashCode();
    result = 31 * result + (realm != null ? realm.hashCode() : 0);
    result = 31 * result + port;
    result = 31 * result + (sslCertificate != null ? sslCertificate.hashCode() : 0);
    result = 31 * result + (sslError != null ? sslError.hashCode() : 0);
    return result;
  }

  @Override
  public String toString() {
    return "URLProtectionSpace{" +
            "host='" + host + '\'' +
            ", protocol='" + protocol + '\'' +
            ", realm='" + realm + '\'' +
            ", port=" + port +
            ", sslCertificate=" + sslCertificate +
            ", sslError=" + sslError +
            '}';
  }
}
