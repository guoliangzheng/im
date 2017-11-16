###1.网络协议
<p>&nbsp;&nbsp;&nbsp;支持tcp(tls)和websocket访问(可以同时使用两种provider)</p>

<br />

###2.通信协议
<p>&nbsp;&nbsp;&nbsp;mqtt3.1.1，在标准化的同时，进行了自定义，以便进行更灵活、更复杂的控制，所以接入时一定要看文档</p>

<br />

###3.编码

**编码对象**

&nbsp;&nbsp;&nbsp;mqtt publish包的payload(payload的内容是我们真正要传递的消息内容，比如你想告诉对方hello world，那么payload="hello world")

**编码格式：PlainText和Json**

- PlainText是文本模式，发送aaaaa，那么收到者就是aaaaa，不做任何编码
- 选用Json格式的原因：
- 支持更多样化的消息类型和内容控制
- 具有良好的可读性

<br />
###4.压缩
publish payload可以选用压缩方式和级别，通过json最外层的字段控制,压缩后的内容是在最外层Json的一个字段中存储

<br />

###5.后续更新
更多的编码：
- protobuf
- messagepack
更多的网络协议：
- http
- http2