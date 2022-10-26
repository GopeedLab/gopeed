import 'package:json_annotation/json_annotation.dart';

part 'start_config.g.dart';

@JsonSerializable()
class StartConfig {
  String network;
  String address;
  String storage;
  String storageDir;
  int refreshInterval;

  StartConfig({
    required this.network,
    required this.address,
    required this.storage,
    required this.storageDir,
    required this.refreshInterval,
  });

  factory StartConfig.fromJson(Map<String, dynamic> json) =>
      _$StartConfigFromJson(json);

  Map<String, dynamic> toJson() => _$StartConfigToJson(this);
}
