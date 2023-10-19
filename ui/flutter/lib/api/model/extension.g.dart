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
      repository: json['repository'] == null
          ? null
          : Repository.fromJson(json['repository'] as Map<String, dynamic>),
      disabled: json['disabled'] as bool,
      devMode: json['devMode'] as bool,
      devPath: json['devPath'] as String,
    )..settings = (json['settings'] as List<dynamic>?)
        ?.map((e) => Setting.fromJson(e as Map<String, dynamic>))
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
  };

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('repository', instance.repository?.toJson());
  writeNotNull('settings', instance.settings?.map((e) => e.toJson()).toList());
  val['disabled'] = instance.disabled;
  val['devMode'] = instance.devMode;
  val['devPath'] = instance.devPath;
  return val;
}

Repository _$RepositoryFromJson(Map<String, dynamic> json) => Repository(
      url: json['url'] as String,
      directory: json['directory'] as String,
    );

Map<String, dynamic> _$RepositoryToJson(Repository instance) =>
    <String, dynamic>{
      'url': instance.url,
      'directory': instance.directory,
    };

Setting _$SettingFromJson(Map<String, dynamic> json) => Setting(
      name: json['name'] as String,
      title: json['title'] as String,
      description: json['description'] as String,
      required: json['required'] as bool,
      type: $enumDecode(_$SettingTypeEnumMap, json['type']),
    )
      ..value = json['value']
      ..options = (json['options'] as List<dynamic>?)
          ?.map((e) => Option.fromJson(e as Map<String, dynamic>))
          .toList();

Map<String, dynamic> _$SettingToJson(Setting instance) {
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

  writeNotNull('value', instance.value);
  writeNotNull('options', instance.options);
  return val;
}

const _$SettingTypeEnumMap = {
  SettingType.string: 'string',
  SettingType.number: 'number',
  SettingType.boolean: 'boolean',
};

Option _$OptionFromJson(Map<String, dynamic> json) => Option(
      label: json['label'] as String,
      value: json['value'] as Object,
    );

Map<String, dynamic> _$OptionToJson(Option instance) => <String, dynamic>{
      'label': instance.label,
      'value': instance.value,
    };
