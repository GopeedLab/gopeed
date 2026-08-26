import 'package:json_annotation/json_annotation.dart';

part 'update_check_extension_resp.g.dart';

@JsonSerializable()
class UpdateCheckExtensionResp {
  String newVersion;

  UpdateCheckExtensionResp({
    required this.newVersion,
  });

  factory UpdateCheckExtensionResp.fromJson(Map<String, dynamic> json) =>
      _$UpdateCheckExtensionRespFromJson(json);
  Map<String, dynamic> toJson() => _$UpdateCheckExtensionRespToJson(this);
}
