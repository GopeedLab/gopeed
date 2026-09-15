package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import io.flutter.plugin.common.MethodChannel;

public interface ICallbackResult<T> extends MethodChannel.Result {
  boolean nonNullSuccess(@NonNull T result);
  boolean nullSuccess();
  void defaultBehaviour(@Nullable T result);
  @Nullable T decodeResult(@Nullable Object obj);
}
