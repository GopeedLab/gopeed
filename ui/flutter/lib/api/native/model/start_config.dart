import 'package:json_annotation/json_annotation.dart';

part 'start_config.g.dart';

@JsonSerializable()
class StartConfig {
  String storage;
  String storageDir;

  StartConfig({
    this.storage = '',
    this.storageDir = '',
  });

  factory StartConfig.fromJson(Map<String, dynamic> json) =>
      _$StartConfigFromJson(json);

  Map<String, dynamic> toJson() => _$StartConfigToJson(this);
}
