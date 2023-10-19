import 'package:flutter/material.dart';
import 'package:gopeed/api/model/history_data.dart';
import 'package:gopeed/app/modules/history/controller/history_controller.dart';

class HistoryView extends StatefulWidget {
  const HistoryView({super.key});

  @override
  State<HistoryView> createState() => _HistoryViewState();
}

class _HistoryViewState extends State<HistoryView> {
  List<HistoryData> listOfHistoryData = [];
  HistoryController _controller = HistoryController();
  @override
  void initState() {
    super.initState();
    getAllHistory();
  }

  void getAllHistory() async {
    List<HistoryData> resultOfHistories = await _controller.getAllHistory();
    setState(() {
      listOfHistoryData = resultOfHistories;
    });
  }

  @override
  Widget build(BuildContext context) {
    final Size size = MediaQuery.sizeOf(context);
    return Scaffold(
      backgroundColor: Colors.transparent,
      body: Dialog(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
        elevation: 0,
        backgroundColor: Colors.transparent,
        child: Container(
          width: size.width * 0.8,
          height: size.height * 0.8,
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(10.0),
          ),
          child: Center(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: listOfHistoryData.isEmpty
                  ? const <Widget>[
                      Icon(
                        Icons.manage_history_rounded,
                      ),
                      SizedBox(
                        height: 10.0,
                      ),
                      Text(
                        "No History Found",
                      ),
                    ]
                  : <Widget>[
                      Expanded(
                        child: ListView.builder(
                          itemBuilder: (context, index) {
                            return ListTile(
                              title: Text(
                                listOfHistoryData[index].text.toString(),
                              ),
                            );
                          },
                        ),
                      ),
                    ],
            ),
          ),
        ),
      ),
    );
  }
}
