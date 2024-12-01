import 'package:json_annotation/json_annotation.dart';

part 'http_listen_result.g.dart';

@JsonSerializable()
class HttpListenResult {
  String host;
  int port;

  HttpListenResult({
    this.host = '',
    this.port = 0,
  });

  factory HttpListenResult.fromJson(Map<String, dynamic> json) =>
      _$HttpListenResultFromJson(json);
  Map<String, dynamic> toJson() => _$HttpListenResultToJson(this);
}
