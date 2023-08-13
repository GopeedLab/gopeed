// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'extension.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Extension _$ExtensionFromJson(Map<String, dynamic> json) => Extension(
      identity: json['identity'] as String,
      name: json['name'] as String,
      author: json['author'] as String,
      title: json['title'] as String,
      description: json['description'] as String,
      icon: json['icon'] as String,
      version: json['version'] as String,
      homepage: json['homepage'] as String,
      installUrl: json['installUrl'] as String,
      repository: json['repository'] as String,
      disabled: json['disabled'] as bool,
    )..settings = (json['settings'] as List<dynamic>?)
        ?.map((e) => Settings.fromJson(e as Map<String, dynamic>))
        .toList();

Map<String, dynamic> _$ExtensionToJson(Extension instance) {
  final val = <String, dynamic>{
    'identity': instance.identity,
    'name': instance.name,
    'author': instance.author,
    'title': instance.title,
    'description': instance.description,
    'icon': instance.icon,
    'version': instance.version,
    'homepage': instance.homepage,
    'installUrl': instance.installUrl,
    'repository': instance.repository,
  };

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('settings', instance.settings?.map((e) => e.toJson()).toList());
  val['disabled'] = instance.disabled;
  return val;
}

Settings _$SettingsFromJson(Map<String, dynamic> json) => Settings(
      name: json['name'] as String,
      title: json['title'] as String,
      description: json['description'] as String,
      required: json['required'] as bool,
      type: $enumDecode(_$SettingTypeEnumMap, json['type']),
      multiple: json['multiple'] as bool,
    )
      ..defaultValue = json['defaultValue']
      ..value = json['value']
      ..options = (json['options'] as List<dynamic>?)
          ?.map((e) => Option.fromJson(e as Map<String, dynamic>))
          .toList();

Map<String, dynamic> _$SettingsToJson(Settings instance) {
  final val = <String, dynamic>{
    'name': instance.name,
    'title': instance.title,
    'description': instance.description,
    'required': instance.required,
    'type': _$SettingTypeEnumMap[instance.type]!,
  };

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('defaultValue', instance.defaultValue);
  writeNotNull('value', instance.value);
  val['multiple'] = instance.multiple;
  writeNotNull('options', instance.options);
  return val;
}

const _$SettingTypeEnumMap = {
  SettingType.string: 'string',
  SettingType.number: 'number',
  SettingType.boolean: 'boolean',
};

Option _$OptionFromJson(Map<String, dynamic> json) => Option(
      title: json['title'] as String,
      value: json['value'] as Object,
    );

Map<String, dynamic> _$OptionToJson(Option instance) => <String, dynamic>{
      'title': instance.title,
      'value': instance.value,
    };
