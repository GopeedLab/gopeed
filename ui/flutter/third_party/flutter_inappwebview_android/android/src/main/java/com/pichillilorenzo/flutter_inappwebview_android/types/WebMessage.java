package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.Nullable;

import java.util.List;

public class WebMessage {
  @Nullable
  public String data;
  @Nullable
  public List<WebMessagePort> ports;

  public WebMessage(@Nullable String data, @Nullable List<WebMessagePort> ports) {
    this.data = data;
    this.ports = ports;
  }

  public void dispose() {
    if (ports != null) {
      ports.clear();
    }
  }
}
