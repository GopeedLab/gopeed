package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

public class BaseCallbackResultImpl<T> implements ICallbackResult<T> {
  @Override
  public boolean nonNullSuccess(@NonNull T result) {
    return true;
  }

  @Override
  public boolean nullSuccess() {
    return true;
  }

  @Override
  public void defaultBehaviour(@Nullable T result) {}

  @Override
  public void success(@Nullable Object obj) {
    T result = decodeResult(obj);
    boolean shouldRunDefaultBehaviour;
    if (result == null) {
      shouldRunDefaultBehaviour = nullSuccess();
    } else {
      shouldRunDefaultBehaviour = nonNullSuccess(result);
    }
    if (shouldRunDefaultBehaviour) {
      defaultBehaviour(result);
    }
  }

  @Nullable
  @Override
  public T decodeResult(@Nullable Object obj) {
    return null;
  }

  @Override
  public void error(String errorCode, @Nullable String errorMessage, @Nullable Object errorDetails) {}

  @Override
  public void notImplemented() {
    defaultBehaviour(null);
  }
}
