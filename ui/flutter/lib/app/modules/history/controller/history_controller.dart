import 'package:gopeed/api/model/history_data.dart';
import 'package:isar/isar.dart';
import 'package:path_provider/path_provider.dart';

class HistoryController {
  Future<List<HistoryData>> getAllHistory() async {
    final dir = await getApplicationDocumentsDirectory();

    final isar = await Isar.open(
      [HistoryDataSchema],
      directory: dir.path,
    );
    List<HistoryData> data = await isar.historyDatas.where().findAll();
    await isar.close();
    return data;
  }

  void addHistory(HistoryData data) async {
    final dir = await getApplicationDocumentsDirectory();

    final isar = await Isar.open(
      [HistoryDataSchema],
      directory: dir.path,
    );

    await isar.writeTxn(() async {
      await isar.historyDatas.put(data); 
    });
    await isar.close();
  }
}
