import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:gopeed/api/model/task.dart';

class TaskController extends GetxController {
  final tabIndex = 0.obs;
  final scaffoldKey = GlobalKey<ScaffoldState>();
  final selectTask = Rx<Task?>(null);
  final copyUrlDone = false.obs;
}
