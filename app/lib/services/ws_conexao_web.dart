import 'package:web_socket_channel/web_socket_channel.dart';

WebSocketChannel conectarWebSocket(Uri uri) => WebSocketChannel.connect(uri);
