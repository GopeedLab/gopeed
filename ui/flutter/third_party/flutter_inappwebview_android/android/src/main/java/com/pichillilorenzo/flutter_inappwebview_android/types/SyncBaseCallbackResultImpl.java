package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.CallSuper;
import androidx.annotation.Nullable;

import java.util.concurrent.CountDownLatch;

public class SyncBaseCallbackResultImpl<T> extends BaseCallbackResultImpl<T> {
  final public CountDownLatch latch = new CountDownLatch(1);
  @Nullable
  public T result = null;

  @CallSuper
  @Override
  public void defaultBehaviour(@Nullable T result) {
    latch.countDown();
  }

  @Override
  public void success(@Nullable Object obj) {
    T result = decodeResult(obj);
    this.result = result;
    boolean shouldRunDefaultBehaviour;
    if (result == null) {
      shouldRunDefaultBehaviour = nullSuccess();
    } else {
      shouldRunDefaultBehaviour = nonNullSuccess(result);
    }
    if (shouldRunDefaultBehaviour) {
      defaultBehaviour(result);
    } else {
      latch.countDown();
    }
  }
  
  @CallSuper
  @Override
  public void error(String errorCode, @Nullable String errorMessage, @Nullable Object errorDetails) {
    latch.countDown();
  }

  @CallSuper
  @Override
  public void notImplemented() {
    defaultBehaviour(null);
  }
}
