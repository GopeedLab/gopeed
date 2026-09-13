package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import java.util.HashMap;
import java.util.Map;

public class DownloadStartRequest {
  
  @NonNull
  private String url;
  @NonNull
  private String userAgent;
  @NonNull
  private String contentDisposition;
  @NonNull
  private String mimeType;
  private long contentLength;
  @Nullable
  private String suggestedFilename;
  @Nullable
  private String textEncodingName;

  public DownloadStartRequest(@NonNull String url, @NonNull String userAgent, @NonNull String contentDisposition, @NonNull String mimeType, long contentLength, @Nullable String suggestedFilename, @Nullable String textEncodingName) {
    this.url = url;
    this.userAgent = userAgent;
    this.contentDisposition = contentDisposition;
    this.mimeType = mimeType;
    this.contentLength = contentLength;
    this.suggestedFilename = suggestedFilename;
    this.textEncodingName = textEncodingName;
  }

  public Map<String, Object> toMap() {
    Map<String, Object> obj = new HashMap<>();
    obj.put("url", url);
    obj.put("userAgent", userAgent);
    obj.put("contentDisposition", contentDisposition);
    obj.put("mimeType", mimeType);
    obj.put("contentLength", contentLength);
    obj.put("suggestedFilename", suggestedFilename);
    obj.put("textEncodingName", textEncodingName);
    return obj;
  }

  @NonNull
  public String getUrl() {
    return url;
  }

  public void setUrl(@NonNull String url) {
    this.url = url;
  }

  @NonNull
  public String getUserAgent() {
    return userAgent;
  }

  public void setUserAgent(@NonNull String userAgent) {
    this.userAgent = userAgent;
  }

  @NonNull
  public String getContentDisposition() {
    return contentDisposition;
  }

  public void setContentDisposition(@NonNull String contentDisposition) {
    this.contentDisposition = contentDisposition;
  }

  @NonNull
  public String getMimeType() {
    return mimeType;
  }

  public void setMimeType(@NonNull String mimeType) {
    this.mimeType = mimeType;
  }

  public long getContentLength() {
    return contentLength;
  }

  public void setContentLength(long contentLength) {
    this.contentLength = contentLength;
  }

  @Nullable
  public String getSuggestedFilename() {
    return suggestedFilename;
  }

  public void setSuggestedFilename(@Nullable String suggestedFilename) {
    this.suggestedFilename = suggestedFilename;
  }

  @Nullable
  public String getTextEncodingName() {
    return textEncodingName;
  }

  public void setTextEncodingName(@Nullable String textEncodingName) {
    this.textEncodingName = textEncodingName;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    DownloadStartRequest that = (DownloadStartRequest) o;

    if (contentLength != that.contentLength) return false;
    if (!url.equals(that.url)) return false;
    if (!userAgent.equals(that.userAgent)) return false;
    if (!contentDisposition.equals(that.contentDisposition)) return false;
    if (!mimeType.equals(that.mimeType)) return false;
    if (suggestedFilename != null ? !suggestedFilename.equals(that.suggestedFilename) : that.suggestedFilename != null)
      return false;
    return textEncodingName != null ? textEncodingName.equals(that.textEncodingName) : that.textEncodingName == null;
  }

  @Override
  public int hashCode() {
    int result = url.hashCode();
    result = 31 * result + userAgent.hashCode();
    result = 31 * result + contentDisposition.hashCode();
    result = 31 * result + mimeType.hashCode();
    result = 31 * result + (int) (contentLength ^ (contentLength >>> 32));
    result = 31 * result + (suggestedFilename != null ? suggestedFilename.hashCode() : 0);
    result = 31 * result + (textEncodingName != null ? textEncodingName.hashCode() : 0);
    return result;
  }

  @Override
  public String toString() {
    return "DownloadStartRequest{" +
            "url='" + url + '\'' +
            ", userAgent='" + userAgent + '\'' +
            ", contentDisposition='" + contentDisposition + '\'' +
            ", mimeType='" + mimeType + '\'' +
            ", contentLength=" + contentLength +
            ", suggestedFilename='" + suggestedFilename + '\'' +
            ", textEncodingName='" + textEncodingName + '\'' +
            '}';
  }
}
