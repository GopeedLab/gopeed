import 'package:json_annotation/json_annotation.dart';

part 'entity.g.dart';

@JsonSerializable()
class StartConfigEntity {
  String network;
  String address;
  String apiToken;

  StartConfigEntity({
    required this.network,
    required this.address,
    required this.apiToken,
  });

  factory StartConfigEntity.fromJson(Map<String, dynamic> json) =>
      _$StartConfigEntityFromJson(json);
  Map<String, dynamic> toJson() => _$StartConfigEntityToJson(this);
}

@JsonSerializable()
class WindowStateEntity {
  bool? isMaximized;
  double? width;
  double? height;

  WindowStateEntity({
    this.isMaximized,
    this.width,
    this.height,
  });

  factory WindowStateEntity.fromJson(Map<String, dynamic> json) =>
      _$WindowStateEntityFromJson(json);
  Map<String, dynamic> toJson() => _$WindowStateEntityToJson(this);
}
