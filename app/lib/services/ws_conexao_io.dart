import 'package:web_socket_channel/io.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

WebSocketChannel conectarWebSocket(Uri uri) => IOWebSocketChannel.connect(
  uri,
  pingInterval: const Duration(seconds: 10),
  connectTimeout: const Duration(seconds: 10),
);
