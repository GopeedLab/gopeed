import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:url_launcher/url_launcher.dart';

class OpenInNew extends StatelessWidget {
  final String text;
  final String url;

  const OpenInNew({super.key, required this.text, required this.url});

  @override
  Widget build(BuildContext context) {
    return Stack(
      children: [
        ElevatedButton(
          onPressed: () {
            launchUrl(Uri.parse(url), mode: LaunchMode.externalApplication);
          },
          style: ElevatedButton.styleFrom(
            backgroundColor: Get.theme.colorScheme.background,
          ),
          child: Row(
            mainAxisSize: MainAxisSize
                .min, // Set the row's size to be as small as possible
            children: <Widget>[
              Text(text),
              const SizedBox(
                  width: 4), // Add some space between the text and the icon
              const Icon(
                Icons.open_in_new,
                size: 14,
              ), // The icon is after the text
            ],
          ),
        )
      ],
    );
  }
}
