import 'package:json_annotation/json_annotation.dart';

part 'update_extension_settings.g.dart';

@JsonSerializable()
class UpdateExtensionSettings {
  Map<String, dynamic> settings;

  UpdateExtensionSettings({
    required this.settings,
  });

  factory UpdateExtensionSettings.fromJson(Map<String, dynamic> json) =>
      _$UpdateExtensionSettingsFromJson(json);
  Map<String, dynamic> toJson() => _$UpdateExtensionSettingsToJson(this);
}
