├── chain
│ ├── chain.go
│ └── chain_test.go
├── config.go
├── engine.go //保存整个 http 服务的路由、中间件信息、并启动 http server：ng.routes->router && start
├── engine_test.go
├── handler
│ ├── authhandler.go
│ ├── authhandler_test.go
│ ├── breakerhandler.go
│ ├── breakerhandler_test.go
│ ├── contentsecurityhandler.go
│ ├── contentsecurityhandler_test.go
│ ├── cryptionhandler.go
│ ├── cryptionhandler_test.go
│ ├── gunziphandler.go
│ ├── gunziphandler_test.go
│ ├── loghandler.go
│ ├── loghandler_test.go
│ ├── maxbyteshandler.go
│ ├── maxbyteshandler_test.go
│ ├── maxconnshandler.go
│ ├── maxconnshandler_test.go
│ ├── metrichandler.go
│ ├── metrichandler_test.go
│ ├── prometheushandler.go
│ ├── prometheushandler_test.go
│ ├── recoverhandler.go
│ ├── recoverhandler_test.go
│ ├── sheddinghandler.go
│ ├── sheddinghandler_test.go
│ ├── timeouthandler.go
│ ├── timeouthandler_test.go
│ ├── tracehandler.go
│ └── tracehandler_test.go
├── httpc
│ ├── internal
│ │ ├── interceptor.go
│ │ ├── loginterceptor.go
│ │ ├── loginterceptor_test.go
│ │ ├── metricsinterceptor.go
│ │ └── metricsinterceptor_test.go
│ ├── requests.go
│ ├── requests_test.go
│ ├── responses.go
│ ├── responses_test.go
│ ├── service.go
│ ├── service_test.go
│ └── vars.go
├── httpx
│ ├── requests.go
│ ├── requests_test.go
│ ├── responses.go
│ ├── responses_test.go
│ ├── router.go
│ ├── util.go
│ ├── util_test.go
│ └── vars.go
├── internal
│ ├── cors
│ │ ├── handlers.go
│ │ └── handlers_test.go
│ ├── encoding
│ │ ├── parser.go
│ │ └── parser_test.go
│ ├── errcode
│ │ ├── grpc.go
│ │ └── grpc_test.go
│ ├── header
│ │ └── headers.go
│ ├── log.go
│ ├── log_test.go
│ ├── response
│ │ ├── headeronceresponsewriter.go
│ │ ├── headeronceresponsewriter_test.go
│ │ ├── withcoderesponsewriter.go
│ │ └── withcoderesponsewriter_test.go
│ ├── security
│ │ ├── contentsecurity.go
│ │ └── contentsecurity_test.go
│ ├── starter.go
│ └── starter_test.go
├── pathvar
│ ├── params.go
│ └── params_test.go
├── readme.md
├── router
│ ├── patrouter.go
│ └── patrouter_test.go
├── server.go
├── server_test.go
├── token
│ ├── tokenparser.go
│ └── tokenparser_test.go
└── types.go
