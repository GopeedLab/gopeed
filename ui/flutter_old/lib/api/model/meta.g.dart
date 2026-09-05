// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'meta.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Meta _$MetaFromJson(Map<String, dynamic> json) => Meta(
      req: Request.fromJson(json['req'] as Map<String, dynamic>),
      opts: Options.fromJson(json['opts'] as Map<String, dynamic>),
    )..res = json['res'] == null
        ? null
        : Resource.fromJson(json['res'] as Map<String, dynamic>);

Map<String, dynamic> _$MetaToJson(Meta instance) {
  final val = <String, dynamic>{
    'req': instance.req.toJson(),
  };

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('res', instance.res?.toJson());
  val['opts'] = instance.opts.toJson();
  return val;
}
