import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:gopeed/database/database.dart';

class HistoryView extends StatefulWidget {
  const HistoryView({
    super.key,
    required this.isHistoryListEmpty,
    required this.historyList,
  });

  final bool isHistoryListEmpty;
  final Widget historyList;

  @override
  State<HistoryView> createState() => _HistoryViewState();
}

class _HistoryViewState extends State<HistoryView> {
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
            color: Theme.of(context).colorScheme.surface,
            borderRadius: BorderRadius.circular(10.0),
          ),
          child: Column(
            children: <Widget>[
              Padding(
                padding: const EdgeInsets.all(8.0),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  crossAxisAlignment: CrossAxisAlignment.center,
                  children: <Widget>[
                    Row(
                      mainAxisAlignment: MainAxisAlignment.start,
                      children: <Widget>[
                        Icon(
                          Icons.history_rounded,
                          size: Theme.of(context)
                              .textTheme
                              .headlineSmall
                              ?.fontSize,
                        ),
                        const SizedBox(width: 8.0),
                        Text(
                          'history'.tr,
                          style: Theme.of(context).textTheme.headlineSmall,
                        ),
                      ],
                    ),
                    Row(
                      children: <Widget>[
                        IconButton(
                          onPressed: () {
                            Database.instance.clearCreateHistory();
                            Navigator.pop(context);
                          },
                          tooltip: "clearHistory".tr,
                          icon: const Icon(
                            Icons.history_toggle_off_rounded,
                          ),
                        ),
                        IconButton(
                          onPressed: () => Navigator.pop(context),
                          icon: const Icon(
                            Icons.close_rounded,
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
              Expanded(
                child: Center(
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: widget.isHistoryListEmpty
                        ? <Widget>[
                            const Icon(
                              Icons.manage_history_rounded,
                            ),
                            const SizedBox(height: 10.0),
                            Text(
                              'noHistoryFound'.tr,
                            ),
                          ]
                        : <Widget>[
                            Expanded(child: widget.historyList),
                          ],
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
