import 'package:shared_preferences/shared_preferences.dart';

class HistoryController {
  // Get History in List<String>
  Future<List<String>> getAllHistory() async {
    SharedPreferences preferences = await SharedPreferences.getInstance();
    List<String>? historyList = preferences.getStringList('history');
    if(historyList == null) {
      return [];
    }
    return historyList;
  }

  // Get Existing History & Add New History
  void addHistory(String historyText) async {
    SharedPreferences preferences = await SharedPreferences.getInstance();
    List<String> existingHistoryData = await getAllHistory();
    existingHistoryData.add(historyText);
    preferences.setStringList('history', existingHistoryData);
  }

  void clearHistory() async {
    SharedPreferences preferences = await SharedPreferences.getInstance();
    preferences.remove('history');
  }
}
