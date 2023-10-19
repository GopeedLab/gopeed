import 'package:isar/isar.dart';

part 'history_data.g.dart';

@collection
class HistoryData {
  Id id = Isar.autoIncrement;
  String? text;
}