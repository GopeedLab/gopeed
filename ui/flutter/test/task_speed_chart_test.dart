import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/app/views/task_speed_chart.dart';

void main() {
  group('TaskSpeedChart Widget Tests', () {
    testWidgets('1. renders without error given an empty list',
        (WidgetTester tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: Center(
              child: TaskSpeedChart(
                speedSamples: [],
                height: 100,
                width: 300,
              ),
            ),
          ),
        ),
      );

      expect(find.byType(TaskSpeedChart), findsOneWidget);
      expect(find.byType(CustomPaint), findsWidgets);
      expect(tester.takeException(), isNull);
    });

    testWidgets('1b. renders custom empty placeholder if provided on empty list',
        (WidgetTester tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: Center(
              child: TaskSpeedChart(
                speedSamples: [],
                height: 100,
                width: 300,
                emptyPlaceholder: Text('No speed data'),
              ),
            ),
          ),
        ),
      );

      expect(find.text('No speed data'), findsOneWidget);
      expect(tester.takeException(), isNull);
    });

    testWidgets('2. renders without error given a single data point',
        (WidgetTester tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: Center(
              child: TaskSpeedChart(
                speedSamples: [1024000.0],
                height: 100,
                width: 300,
              ),
            ),
          ),
        ),
      );

      expect(find.byType(TaskSpeedChart), findsOneWidget);
      expect(find.byType(CustomPaint), findsWidgets);
      expect(tester.takeException(), isNull);
    });

    testWidgets('3. renders correctly given a typical list of 60 data points',
        (WidgetTester tester) async {
      final samples = List.generate(60, (index) => (index * 50000).toDouble());

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Center(
              child: TaskSpeedChart(
                speedSamples: samples,
                height: 120,
                width: 350,
              ),
            ),
          ),
        ),
      );

      expect(find.byType(TaskSpeedChart), findsOneWidget);
      expect(find.byType(CustomPaint), findsWidgets);
      expect(tester.takeException(), isNull);
    });

    testWidgets(
        '4. does not throw on rapidly changing data (simulating multiple rebuilds)',
        (WidgetTester tester) async {
      List<double> currentSamples = [];

      await tester.pumpWidget(
        StatefulBuilder(
          builder: (context, setState) {
            return MaterialApp(
              home: Scaffold(
                body: Center(
                  child: TaskSpeedChart(
                    speedSamples: currentSamples,
                    height: 100,
                    width: 300,
                  ),
                ),
              ),
            );
          },
        ),
      );
      expect(tester.takeException(), isNull);

      // Simulate 60 rapid data arrivals
      for (int i = 1; i <= 60; i++) {
        currentSamples = List.generate(i, (idx) => (idx * 1024 * 10).toDouble());
        await tester.pumpWidget(
          MaterialApp(
            home: Scaffold(
              body: Center(
                child: TaskSpeedChart(
                  speedSamples: currentSamples,
                  height: 100,
                  width: 300,
                ),
              ),
            ),
          ),
        );
        expect(tester.takeException(), isNull);
      }

      // Simulate sudden drop or reset
      currentSamples = [];
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Center(
              child: TaskSpeedChart(
                speedSamples: currentSamples,
                height: 100,
                width: 300,
              ),
            ),
          ),
        ),
      );
      expect(tester.takeException(), isNull);
    });
  });
}
