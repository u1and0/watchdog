`watchdog` is a continuous health monitoring CLI tool

Usage:
  health-monitor [flags]

Flags:
  -e, --endpoint string         Target endpoint to monitor
  -h, --help                    help for health-monitor
  -M, --max-interval duration   Maximum check interval (default 1h0m0s)
  -m, --min-interval duration   Minimum check interval (default 1m0s)
  -u, --user string             Slack user ID to mention in error messages
  -w, --webhook string          Slack webhook URL

```
# minimal option
$ watchdog -e http://localhost:8080 -w $SLACK_WEBHOOK
```


```
# full option
$ watchdog -e http://localhost:8080 -w $SLACK_WEBHOOK -m 2s -M 16s -u $SLACK_USER
```

上記オプションでは、
最小間隔2秒、最大間隔16秒ごとにendpointに指定したURLへアクセスを試みます。
エラー発生時には、webhookに指定したチャンネルへ、userで指定したユーザー向けにリプライを飛ばします。
エラー発生時は指数バックオフの考えを使って、リトライ間隔を最大間隔まで伸ばします。
