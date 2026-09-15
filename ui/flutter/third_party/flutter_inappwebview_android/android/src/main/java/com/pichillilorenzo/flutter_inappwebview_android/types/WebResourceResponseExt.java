package com.pichillilorenzo.flutter_inappwebview_android.types;

import android.os.Build;
import android.webkit.WebResourceResponse;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import com.pichillilorenzo.flutter_inappwebview_android.Util;

import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;

public class WebResourceResponseExt {
  @Nullable
  private String contentType;
  @Nullable
  private String contentEncoding;
  @Nullable
  private Integer statusCode;
  @Nullable
  private String reasonPhrase;
  @Nullable
  private Map<String, String> headers;
  @Nullable
  private byte[] data;

  public WebResourceResponseExt(@Nullable String contentType, @Nullable String contentEncoding, @Nullable Integer statusCode,
                                @Nullable String reasonPhrase, @Nullable Map<String, String> headers, @Nullable byte[] data) {
    this.contentType = contentType;
    this.contentEncoding = contentEncoding;
    this.statusCode = statusCode;
    this.reasonPhrase = reasonPhrase;
    this.headers = headers;
    this.data = data;
  }

  static public WebResourceResponseExt fromWebResourceResponse(@NonNull WebResourceResponse response) {
    Integer statusCode = null;
    String reasonPhrase = null;
    Map<String, String> headers = null;
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
      statusCode = response.getStatusCode();
      reasonPhrase = response.getReasonPhrase();
      headers = response.getResponseHeaders();
    }
    return new WebResourceResponseExt(response.getMimeType(),
            response.getEncoding(),
            statusCode,
            reasonPhrase,
            headers,
            Util.readAllBytes(response.getData())
    );
  }

  @Nullable
  public static WebResourceResponseExt fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    String contentType = (String) map.get("contentType");
    String contentEncoding = (String) map.get("contentEncoding");
    Integer statusCode = (Integer) map.get("statusCode");
    String reasonPhrase = (String) map.get("reasonPhrase");
    Map<String, String> headers = (Map<String, String>) map.get("headers");
    byte[] data = (byte[]) map.get("data");
    return new WebResourceResponseExt(contentType, contentEncoding, statusCode, reasonPhrase, headers, data);
  }

  public Map<String, Object> toMap() {
    Map<String, Object> webResourceResponseMap = new HashMap<>();
    webResourceResponseMap.put("contentType", contentType);
    webResourceResponseMap.put("contentEncoding", contentEncoding);
    webResourceResponseMap.put("statusCode", statusCode);
    webResourceResponseMap.put("reasonPhrase", reasonPhrase);
    webResourceResponseMap.put("headers", headers);
    webResourceResponseMap.put("data", data);
    return webResourceResponseMap;
  }

  @Nullable
  public String getContentType() {
    return contentType;
  }

  public void setContentType(@Nullable String contentType) {
    this.contentType = contentType;
  }

  @Nullable
  public String getContentEncoding() {
    return contentEncoding;
  }

  public void setContentEncoding(@Nullable String contentEncoding) {
    this.contentEncoding = contentEncoding;
  }

  @Nullable
  public Integer getStatusCode() {
    return statusCode;
  }

  public void setStatusCode(@Nullable Integer statusCode) {
    this.statusCode = statusCode;
  }

  @Nullable
  public String getReasonPhrase() {
    return reasonPhrase;
  }

  public void setReasonPhrase(@Nullable String reasonPhrase) {
    this.reasonPhrase = reasonPhrase;
  }

  @Nullable
  public Map<String, String> getHeaders() {
    return headers;
  }

  public void setHeaders(@Nullable Map<String, String> headers) {
    this.headers = headers;
  }

  @Nullable
  public byte[] getData() {
    return data;
  }

  public void setData(@Nullable byte[] data) {
    this.data = data;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    WebResourceResponseExt that = (WebResourceResponseExt) o;

    if (contentType != null ? !contentType.equals(that.contentType) : that.contentType != null)
      return false;
    if (contentEncoding != null ? !contentEncoding.equals(that.contentEncoding) : that.contentEncoding != null)
      return false;
    if (statusCode != null ? !statusCode.equals(that.statusCode) : that.statusCode != null)
      return false;
    if (reasonPhrase != null ? !reasonPhrase.equals(that.reasonPhrase) : that.reasonPhrase != null)
      return false;
    if (headers != null ? !headers.equals(that.headers) : that.headers != null) return false;
    return Arrays.equals(data, that.data);
  }

  @Override
  public int hashCode() {
    int result = contentType != null ? contentType.hashCode() : 0;
    result = 31 * result + (contentEncoding != null ? contentEncoding.hashCode() : 0);
    result = 31 * result + (statusCode != null ? statusCode.hashCode() : 0);
    result = 31 * result + (reasonPhrase != null ? reasonPhrase.hashCode() : 0);
    result = 31 * result + (headers != null ? headers.hashCode() : 0);
    result = 31 * result + Arrays.hashCode(data);
    return result;
  }

  @Override
  public String toString() {
    return "WebResourceResponseExt{" +
            "contentType='" + contentType + '\'' +
            ", contentEncoding='" + contentEncoding + '\'' +
            ", statusCode=" + statusCode +
            ", reasonPhrase='" + reasonPhrase + '\'' +
            ", headers=" + headers +
            ", data=" + Arrays.toString(data) +
            '}';
  }
}
