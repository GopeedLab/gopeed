## gopeed
支持多协议（HTTP、BitTorrent）下载的客户端，提供命令行、RESTful API、WebSocket、Go类库方式来使用。

## TODO
- [x] HTTP下载实现
- [ ] BitTorrent下载实现
    - [x] .torrent文件解析
    - [x] tracker协议实现
    - [ ] peer wire protocol协议实现
    - [ ] DHT协议实现
    - [ ] 磁力链接支持
    - [ ] uTP协议实现
- [ ] 下载接口抽象(不关心具体协议)
- [ ] 支持自定义配置
- [ ] 限速功能实现
- [ ] 命令行工具提供
- [ ] RESTful服务提供

## 参与
由于项目目前还未定型，代码可能随时有大的调整，所以暂不接受PR，当然如果有什么好的想法可以在issue区提出来。