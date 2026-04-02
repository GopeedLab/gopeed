import 'package:json_annotation/json_annotation.dart';

part 'start_config.g.dart';

@JsonSerializable()
class WebViewRpcConfig {
  late String network;
  late String address;
  String? token;

  WebViewRpcConfig();

  factory WebViewRpcConfig.fromJson(Map<String, dynamic> json) =>
      _$WebViewRpcConfigFromJson(json);

  Map<String, dynamic> toJson() => _$WebViewRpcConfigToJson(this);
}

@JsonSerializable()
class StartConfig {
  late String network;
  late String address;
  late String storage;
  late String storageDir;
  late int refreshInterval;
  late String apiToken;
  WebViewRpcConfig? webViewRpcConfig;

  StartConfig();

  factory StartConfig.fromJson(Map<String, dynamic> json) =>
      _$StartConfigFromJson(json);

  Map<String, dynamic> toJson() => _$StartConfigToJson(this);
}
