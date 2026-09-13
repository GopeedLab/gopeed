package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.CallSuper;
import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import io.flutter.plugin.common.MethodCall;
import io.flutter.plugin.common.MethodChannel;

public class ChannelDelegateImpl implements IChannelDelegate {
  @Nullable
  private MethodChannel channel;

  public ChannelDelegateImpl(@NonNull MethodChannel channel) {
    this.channel = channel;
    this.channel.setMethodCallHandler(this);
  }
  
  @Override
  @Nullable
  public MethodChannel getChannel() {
    return channel;
  }

  @CallSuper
  public void dispose() {
    if (channel != null) {
      channel.setMethodCallHandler(null);
      channel = null;
    }
  }

  @Override
  public void onMethodCall(@NonNull MethodCall call, @NonNull MethodChannel.Result result) {
    
  }
}
