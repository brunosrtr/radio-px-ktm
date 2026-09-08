import 'package:flutter/material.dart';

void main() {
  runApp(const RadioPxApp());
}

class RadioPxApp extends StatelessWidget {
  const RadioPxApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Rádio PX Digital',
      theme: ThemeData(colorScheme: ColorScheme.fromSeed(seedColor: Colors.deepOrange)),
      home: const Scaffold(body: Center(child: Text('Rádio PX Digital'))),
    );
  }
}
