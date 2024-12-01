import 'package:dio/dio.dart';

import '../api_exception.dart';
import '../model/result.dart';

mixin LibgopeedBootBase {
  Dio createDio() {
    Dio dio = Dio();
    dio.options.contentType = Headers.jsonContentType;
    dio.options.sendTimeout = const Duration(seconds: 5);
    dio.options.connectTimeout = const Duration(seconds: 5);
    dio.options.receiveTimeout = const Duration(seconds: 60);
    return dio;
  }

  T handleResult<T>(
    Result<T> result,
  ) {
    if (result.code == 0) {
      return result.data as T;
    } else {
      throw ApiException(result.code, result.msg ?? '');
    }
  }
}
